package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Cost checks: on a site that sells nothing, its prices are what others
// charge. A rate is a UnitPriceSpecification, a total broken into parts a
// CompoundPriceSpecification, a cost or estimate a MonetaryAmount and goods
// priced across sellers a Product with an AggregateOffer; an Offer is only
// for what the business itself sells. A shop's prices stay its Offers.

const priceText = `(?:A\$|AU\$|US\$|NZ\$|C\$|CA\$|€|£|\$)\s?\d(?:[\d,]*\d)?(?:\.\d{1,2})?`

var (
	priceSpan = `(` + priceText + `(?:\s*(?:–|-|to)\s*` + priceText + `)?)`
	rateRe    = regexp.MustCompile(`(?i)` + priceSpan + `\s*(?:an?|per|/)\s*(?:(mover|person|worker|crew member)\s+(?:an?|per|/)\s*)?(hour|hr|mile|mi|pound|lb|month|day|week|square foot|sq\.? ?ft|cubic foot|foot|ft|box|room|item|load|trip|mover|person)\b`)
	goodsRe   = regexp.MustCompile(`(?i)\b(boxes|box|tape|packing paper|paper|bubble wrap|mattress bags?|moving blankets?|furniture pads?|dollies|dolly|stretch wrap|packing peanuts)\b[^$.]{0,45}?` + priceSpan + `([^.$]{0,25})`)
	goodsSign = regexp.MustCompile(`(?i)\b(?:each|apiece|a pack|per pack|a roll|per roll|published prices|retail prices|typical (?:store|retail) prices)\b`)
	costWords = regexp.MustCompile(`(?i)\b(?:costs?|estimates?|estimated|typical(?:ly)?|average|on average|ranges?|runs?|expect|roughly)\b`)
	goodsNoun = regexp.MustCompile(`(?i)\b(?:boxes|box|tape|packing paper|bubble wrap|mattress bags?|moving blankets?|furniture pads?|dollies|dolly|stretch wrap|packing peanuts)\b`)
	spanRe    = regexp.MustCompile(priceSpan)
	rateUnits = map[string]string{"hour": "HUR", "hr": "HUR", "mile": "SMI", "mi": "SMI", "pound": "LBR", "lb": "LBR", "month": "MON", "day": "DAY", "week": "WEE",
		"foot": "FOT", "ft": "FOT", "square foot": "FTK", "sq ft": "FTK", "sq. ft": "FTK", "sqft": "FTK", "cubic foot": "FTQ"}
	serviceCost = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bcosts?\s+(?:of|to)\s+((?:move|moving|hire|hiring|install|installing|repair|repairing|clean|cleaning|replace|replacing|remove|removing|fix|fixing|build|building|paint|painting|ship|shipping|store|storing|haul|hauling|pack|packing|deliver|delivering)\b[^,:?(]*)`),
		regexp.MustCompile(`(?i)^(?:how much (?:do|does|is|are) |what (?:do|does|is|are) )?(?:the |an? |\d+ )?((?:\w+ )?(?:\w+ers|\w+ing|company|companies|crew|services?|labou?r))\s+(?:cost|costs|charge)\b`),
	}
)

// amounts are the amounts the draft carries as a price or a monetary value.
func (c *checker) amounts(all []located) map[float64]bool {
	out := map[float64]bool{}
	for _, l := range all {
		keys := []string{"price", "lowPrice", "highPrice", "minPrice", "maxPrice"}
		if c.is(l.n, "MonetaryAmount") || c.is(l.n, "PriceSpecification") {
			keys = append(keys, "value", "minValue", "maxValue")
		}
		for _, k := range keys {
			for _, s := range scalars(l.n, k) {
				if f, ok := amount(s); ok {
					out[f] = true
				}
			}
		}
	}
	return out
}

// costs runs the cost checks on a site that sells nothing.
func (c *checker) costs(all []located, p *pageFacts) {
	if p.sells() {
		return
	}
	marked := c.amounts(all)
	// fresh reports text holding an amount neither marked up nor reported
	fresh := func(text string) bool {
		return slices.ContainsFunc(priceAmount.FindAllStringSubmatch(text, -1), func(m []string) bool {
			f, ok := priceOf(m)
			return ok && !marked[f] && !p.claimed[f]
		})
	}
	c.breakdowns(p, fresh)
	c.rates(p, fresh)
	c.goods(p, fresh)
	c.estimates(p, fresh)
	c.service(all, p)
}

// clause is the text before position i in s back to the start of its
// sentence, at most n bytes.
func clause(s string, i, n int) string {
	from := max(0, i-n)
	t := s[from:i]
	if k := strings.LastIndex(t, ". "); k >= 0 {
		t = t[k+2:]
	}
	return t
}

// examples joins up to four examples, and says how many more there are.
func examples(list []string) string {
	if len(list) > 4 {
		return strings.Join(list[:4], ", ") + fmt.Sprintf(" and %d more", len(list)-4)
	}
	return strings.Join(list, ", ")
}

// breakdowns warn on a total the page breaks into priced parts: a list,
// definition list or table of two or more priced lines beside the total (an
// output or data element, or a line labelled total).
func (c *checker) breakdowns(p *pageFacts, fresh func(string) bool) {
	if p.doc == nil {
		return
	}
	for _, list := range queryAll(p.doc, "dl, ul, ol, table") {
		if chrome(list) || !shown(list) {
			continue
		}
		var parts []string
		switch list.DataAtom {
		case atom.Dl:
			for _, d := range pairs(list) {
				if priceAmount.MatchString(d.value) {
					parts = append(parts, d.value)
				}
			}
		case atom.Table:
			for _, tr := range queryAll(list, "tr") {
				if t := htmlText(tr); priceAmount.MatchString(t) && len(queryAll(tr, "td")) > 0 {
					parts = append(parts, t)
				}
			}
		default:
			for _, li := range queryAll(list, "li") {
				if t := htmlText(li); priceAmount.MatchString(t) {
					parts = append(parts, t)
				}
			}
		}
		// a list of goods' prices is the goods check's
		goods := 0
		for _, t := range parts {
			if goodsNoun.MatchString(t) {
				goods++
			}
		}
		if len(parts) < 2 || goods*2 >= len(parts) {
			continue
		}
		// the total sits in the list's own block, at most its article or
		// section
		var box, total *xhtml.Node
		for e, up := list.Parent, 0; e != nil && up < 3 && total == nil && e.DataAtom != atom.Main && e.DataAtom != atom.Body; e, up = e.Parent, up+1 {
			for _, o := range queryAll(e, "output, data[value]") {
				if !slices.Contains(ancestors(o), list) && priceAmount.MatchString(htmlText(o)) {
					box, total = e, o
					break
				}
			}
			if e.DataAtom == atom.Article || e.DataAtom == atom.Section {
				break
			}
		}
		if total == nil || !fresh(htmlText(total)+" "+strings.Join(parts, " ")) {
			continue
		}
		label := "the total"
		for e := box; e != nil && e.DataAtom != atom.Body; e = e.Parent {
			if hs := queryAll(e, "h2, h3, h4, h5"); len(hs) > 0 {
				label = htmlText(hs[0])
				break
			}
		}
		p.claim(htmlText(total) + " " + strings.Join(parts, " "))
		c.add("html:"+selectorFor(p.doc, box), Warning, "the page breaks %q (%s) into %d priced parts: mark it up as a CompoundPriceSpecification (the PriceSpecification that sums its parts: minPrice, maxPrice, priceCurrency) with each part a priceComponent (a UnitPriceSpecification with priceComponentType such as DistanceFee for mileage, or a DeliveryChargeSpecification for a delivery or pickup charge)", label, htmlText(total), len(parts))
	}
}

// rates warn on prices given per unit: an hour, a mile, a pound, per mover.
func (c *checker) rates(p *pageFacts, fresh func(string) bool) {
	var list, codes []string
	perPerson := false
	for _, m := range rateRe.FindAllStringSubmatch(p.main, -1) {
		if !fresh(m[1]) {
			continue
		}
		p.claim(m[1])
		list = append(list, m[0])
		code := rateUnits[strings.ToLower(m[3])]
		if code == "" {
			code = "C62 with unitText " + strings.ToLower(m[3])
		}
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
		perPerson = perPerson || m[2] != ""
	}
	if len(list) == 0 {
		return
	}
	ref := ""
	if perPerson {
		ref = `, and referenceQuantity for a "per mover" rate`
	}
	c.add(pagePath, Warning, "the page prices %s: mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode %s%s) on the Service or Offer it is the rate of", examples(list), strings.Join(codes, " or "), ref)
}

// goods warn on goods the page prices across sellers: boxes, tape, packing
// paper at typical prices.
func (c *checker) goods(p *pageFacts, fresh func(string) bool) {
	var list []string
	for _, m := range goodsRe.FindAllStringSubmatchIndex(p.main, -1) {
		span, after := p.main[m[4]:m[5]], p.main[m[6]:m[7]]
		sentence := clause(p.main, m[0], 200) + p.main[m[0]:m[1]]
		if !fresh(span) || !(goodsSign.MatchString(after) || goodsSign.MatchString(sentence)) {
			continue
		}
		p.claim(span)
		list = append(list, p.main[m[2]:m[3]]+" "+span)
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page prices goods (%s): add a Product for each with an AggregateOffer (lowPrice, highPrice, priceCurrency, and offerCount: how many sellers or sources the range spans)", examples(list))
	}
}

// estimates warn on prices given as costs or estimates of what others
// charge: "a 1-bedroom move costs $380–$960".
func (c *checker) estimates(p *pageFacts, fresh func(string) bool) {
	var list []string
	for _, m := range spanRe.FindAllStringSubmatchIndex(p.main, -1) {
		span := p.main[m[2]:m[3]]
		if !fresh(span) || !costWords.MatchString(clause(p.main, m[0], 80)) {
			continue
		}
		p.claim(span)
		list = append(list, span)
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page estimates costs (%s): mark each up as a MonetaryAmount (value, or minValue and maxValue, currency) on the thing whose cost it is, such as a Service's offers priceSpecification or a PropertyValue, when it is what others charge; give the business's own price as an Offer with seller the business", examples(list))
	}
}

// service warns on a content page about what a service costs ("Cost to move
// a piano", "How much do movers cost?") when no Service describes it.
func (c *checker) service(all []located, p *pageFacts) {
	if p.kind != "" || c.has(all, "Service") {
		return
	}
	h := p.h1()
	for _, re := range serviceCost {
		if m := re.FindStringSubmatch(h); m != nil {
			c.add(pagePath, Warning, "the page is about the cost of the service %q but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity", strings.TrimSpace(m[1]))
			return
		}
	}
}
