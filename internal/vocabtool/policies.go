package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The policy checks hold a shop's markup to the terms its pages state:
// shipping rates, free-shipping thresholds and times, its return policy, how
// to contact it, where it is, when it is open, how to pay, and the questions
// it answers.

const money = `(?:A\$|AU\$|NZ\$|US\$|\$|£|€)\s?(\d+(?:\.\d{1,2})?)`

var (
	freeShipping  = regexp.MustCompile(`(?i)\bfree (?:\w+ ){0,2}(?:shipping|delivery|postage)(?: (?:on|for) (?:all )?orders)?(?: (?:over|above|of|from)| when you spend)\s*` + money)
	shippingRate  = regexp.MustCompile(`(?i)\b(?:shipping|delivery|postage)(?: rate| cost| fee)?(?: within \w+)?(?: is)?(?: a)?(?: flat(?: rate)?(?: of)?)?\s*:?\s*` + money + `|` + money + ` flat[- ]rate (?:shipping|delivery|postage)`)
	handling      = regexp.MustCompile(`(?i)\b(?:posted|dispatched|shipped|ships|sent|processed)(?: out)? within (\d+)(?:\s*(?:-|–|to)\s*(\d+))? (hours?|business days?|working days?|days?)\b|\bprocessing time:?\s*(\d+)\s*(?:-|–|to)\s*(\d+) (business days?|working days?|days?)\b`)
	freeOtherwise = regexp.MustCompile(`(?i)\bfree (?:\w+ ){0,2}(?:over|above|from)\s*` + money + `,?\s*otherwise\s*` + money)
	transit       = regexp.MustCompile(`(?i)\b(?:delivery|transit|arrives?|arrival)(?: time)?(?: in| within)?:?\s*(\d+)\s*(?:-|–|to)\s*(\d+) (business days?|working days?|days?)\b`)
	shipCountry   = regexp.MustCompile(`(?i)\b(?:shipping|delivery|postage|ship|orders?)\s+(?:within|to|across|in)\s+(?:the )?(Australia|AU|New Zealand|NZ|United States|USA|US|United Kingdom|UK|Canada)\b|\bfree (Australian|NZ|US|UK|Canadian) (?:shipping|delivery|postage)\b`)
	countryCodes  = map[string]string{"australia": "AU", "australian": "AU", "au": "AU", "new zealand": "NZ", "nz": "NZ", "united states": "US", "usa": "US", "us": "US",
		"united kingdom": "GB", "uk": "GB", "canada": "CA", "canadian": "CA"}
)

// shippingNodes are the scalars under the draft's shipping nodes.
func (c *checker) shippingScalars(all []located) []string {
	var out []string
	for _, l := range all {
		for _, ty := range []string{"ShippingConditions", "OfferShippingDetails", "ShippingRateSettings", "ShippingService", "DeliveryChargeSpecification"} {
			if c.is(l.n, ty) {
				out = append(out, deepScalars(l.n)...)
				break
			}
		}
	}
	return out
}

// shipping warns on a shipping rate, free-shipping threshold, handling time
// or transit time the page states that no shipping node carries.
func (c *checker) shipping(all []located, p *pageFacts) {
	text := p.text
	ship := c.shippingScalars(all)
	country := "its country"
	if m := shipCountry.FindStringSubmatch(text); m != nil {
		country = countryCodes[strings.ToLower(firstNonEmpty(m[1:]...))]
	}
	if m := freeOtherwise.FindStringSubmatch(text); m != nil {
		text = text + " Shipping rate $" + m[2] + ". Free shipping over $" + m[1] + "."
	}
	if m := shippingRate.FindStringSubmatch(text); m != nil {
		v := firstNonEmpty(m[1:]...)
		f, _ := amount(v)
		p.claimed[f] = true
		if !holdsNumber(ship, f) {
			c.add(pagePath, Warning, "the page gives a shipping rate of $%s that no node carries: add a ShippingService to the business's hasShippingService (fulfillmentType https://schema.org/FulfillmentTypeDelivery) with shippingConditions, a ShippingConditions with shippingDestination (a DefinedRegion with addressCountry %s) and shippingRate (a MonetaryAmount of %s); on a product page give its Offer the same as shippingDetails, an OfferShippingDetails", v, country, v)
		}
	}
	if m := freeShipping.FindStringSubmatch(text); m != nil {
		f, _ := amount(m[1])
		p.claimed[f] = true
		if !holdsNumber(ship, f) {
			c.add(pagePath, Warning, "the page offers free shipping over $%s but no ShippingConditions has an orderValue of %s: add a ShippingConditions with orderValue (a MonetaryAmount with minValue %s) and shippingRate 0 to the business's ShippingService, or a ShippingRateSettings whose freeShippingThreshold is a DeliveryChargeSpecification of %s", m[1], m[1], m[1], m[1])
		}
	}
	handlingVals := under(all, "handlingTime")
	var said []string
	days := 0
	for _, m := range handling.FindAllStringSubmatch(text, -1) {
		lo, hi, unit := m[1], m[2], m[3]
		if m[4] != "" {
			lo, hi, unit = m[4], m[5], m[6]
		}
		maxV, _ := strconv.Atoi(orDefault(hi, lo))
		d := maxV
		if strings.HasPrefix(strings.ToLower(unit), "hour") {
			d = (maxV + 23) / 24
		}
		if !holdsNumber(handlingVals, float64(maxV)) && !holdsNumber(handlingVals, float64(d)) && !slices.Contains(said, m[0]) {
			said, days = append(said, m[0]), max(days, d)
		}
	}
	if len(said) > 0 {
		c.add(pagePath, Warning, "the page says %s but no handlingTime carries it: set handlingTime on the ShippingService (a ServicePeriod, or a QuantitativeValue with maxValue %d and unitCode DAY) or, on a product page, in its OfferShippingDetails's deliveryTime (a ShippingDeliveryTime with handlingTime a QuantitativeValue)", quoteAll(said), days)
	}
	transitVals := under(all, "transitTime")
	if m := transit.FindStringSubmatch(text); m != nil {
		lo, _ := amount(m[1])
		hi, _ := amount(m[2])
		if !holdsNumber(transitVals, lo) || !holdsNumber(transitVals, hi) {
			c.add(pagePath, Warning, "the page says %q but no transitTime carries it: set transitTime on the ShippingConditions (a ServicePeriod with businessDays and a duration of %s to %s days) or in the Offer's ShippingDeliveryTime (a QuantitativeValue with minValue %s, maxValue %s and unitCode DAY)", m[0], m[1], m[2], m[1], m[2])
		}
	}
}

// returnFact is a term of the return policy the page states and the
// property and member that carry it.
type returnFact struct {
	said, prop, member string
}

var (
	returnWindow   = regexp.MustCompile(`(?i)\b(\d+)[- ]day (?:free )?(?:returns?|money[- ]back|refunds?|exchanges?)\b|\breturn(?:ed)?\s+(?:\w+\s+){0,3}?within (\d+) days\b|\b(\d+) days to return\b`)
	returnNone     = regexp.MustCompile(`(?i)\bno returns\b|\breturns (?:are )?not accepted\b|\ball sales (?:are )?final\b|\bnon-?returnable\b`)
	returnSentence = regexp.MustCompile(`(?i)[^.!?]*\b(?:returns?|returned|refunds?|exchanges?)\b[^.!?]*[.!?]?`)
	returnFacts    = []struct {
		re                 *regexp.Regexp
		prop, member, want string
	}{
		{regexp.MustCompile(`(?i)\bfree returns?\b|\breturns? (?:are|is) free\b`), "returnFees", "FreeReturn", ""},
		{regexp.MustCompile(`(?i)\breturn (?:shipping|postage)(?: costs?)? (?:is|are) (?:paid|covered) by (?:the )?(?:customer|buyer|you)\b|\b(?:customer|buyer|you) pays? (?:for )?(?:the )?return`), "returnFees", "ReturnFeesCustomerResponsibility", ""},
		{regexp.MustCompile(`(?i)\brestocking fee\b`), "returnFees", "RestockingFees", ""},
		{regexp.MustCompile(`(?i)\bby (?:mail|post)\b|\bpost (?:it|them|items?) back\b`), "returnMethod", "ReturnByMail", ""},
		{regexp.MustCompile(`(?i)\breturn(?:ed)? (?:it |them |items? )?in[- ]store\b|\bto (?:our|the|any) (?:store|shop)\b`), "returnMethod", "ReturnInStore", ""},
		{regexp.MustCompile(`(?i)\bstore credit\b`), "refundType", "StoreCreditRefund", ""},
		{regexp.MustCompile(`(?i)\bexchanges? only\b|\bfor an exchange\b`), "refundType", "ExchangeRefund", ""},
		{regexp.MustCompile(`(?i)\bfull refund\b|\bmoney back\b`), "refundType", "FullRefund", ""},
		{regexp.MustCompile(`(?i)\b(?:prepaid )?return label (?:is )?(?:included|in the box)\b|\bprepaid return label\b`), "returnLabelSource", "ReturnLabelInBox", ""},
		{regexp.MustCompile(`(?i)\b(?:print|download) (?:a |your |the )?(?:prepaid )?(?:return )?label\b`), "returnLabelSource", "ReturnLabelDownloadAndPrint", ""},
	}
	seasonal    = regexp.MustCompile(`(?i)\b(?:holiday|christmas|seasonal|extended) returns?\b|\bbought between [^.]{3,40} (?:can be|may be) returned until\b`)
	returnsLink = regexp.MustCompile(`(?i)\breturns?\b|\brefunds?\b`)
)

// returns warns on the return policy terms the page states that no
// MerchantReturnPolicy carries, and on a link to the policy page no
// MerchantReturnPolicy names as its merchantReturnLink.
func (c *checker) returns(all []located, p *pageFacts) {
	var facts []returnFact
	if m := returnWindow.FindStringSubmatch(p.text); m != nil {
		days := firstNonEmpty(m[1:]...)
		facts = append(facts, returnFact{m[0], "merchantReturnDays", days}, returnFact{m[0], "returnPolicyCategory", "MerchantReturnFiniteReturnWindow"})
	}
	if m := returnNone.FindString(p.text); m != "" {
		facts = append(facts, returnFact{m, "returnPolicyCategory", "MerchantReturnNotPermitted"})
	}
	for _, s := range returnSentence.FindAllString(p.main, -1) {
		for _, rf := range returnFacts {
			if m := rf.re.FindString(s); m != "" && !slices.ContainsFunc(facts, func(f returnFact) bool { return f.member == rf.member }) {
				facts = append(facts, returnFact{m, rf.prop, rf.member})
			}
		}
	}
	if m := seasonal.FindString(p.main); m != "" {
		facts = append(facts, returnFact{m, "returnPolicySeasonalOverride", "MerchantReturnPolicySeasonalOverride"})
	}
	var policies []located
	for _, l := range all {
		if c.is(l.n, "MerchantReturnPolicy") && len(l.n.keys) > 1 {
			policies = append(policies, l)
		}
	}
	link := ""
	if p.kind == kindReturns && p.url != nil {
		link = normURL(p.url)
	} else if p.doc != nil {
		for _, a := range queryAll(p.doc, "a[href]") {
			if u, ok := p.siteLink(a); ok && returnsLink.MatchString(htmlText(a)) && len(strings.Fields(htmlText(a))) <= 3 {
				link = u
				break
			}
		}
	}
	if len(policies) == 0 {
		if len(facts) == 0 && link == "" {
			return
		}
		var said, set []string
		for _, f := range facts {
			if !slices.Contains(said, f.said) {
				said = append(said, f.said)
			}
			set = append(set, describeReturnFact(f))
		}
		if link != "" {
			set = append(set, "merchantReturnLink "+link)
		}
		what := "links its return policy"
		if len(said) > 0 {
			what = fmt.Sprintf("states a return policy (%s)", quoteAll(said))
		}
		c.add(pagePath, Warning, "the page %s that no MerchantReturnPolicy carries: give the business (and each Offer) hasMerchantReturnPolicy, a MerchantReturnPolicy with applicableCountry, %s", what, strings.Join(set, ", "))
		return
	}
	for _, f := range facts {
		ok := slices.ContainsFunc(policies, func(l located) bool {
			if f.prop == "returnPolicySeasonalOverride" {
				_, has := l.n.get(f.prop)
				return has
			}
			if f.prop == "merchantReturnDays" {
				return slices.Contains(scalars(l.n, f.prop), f.member)
			}
			return slices.ContainsFunc(strs(l.n, f.prop), func(s string) bool { return stripPrefix(s) == f.member })
		})
		if !ok {
			c.add(policies[0].path, Warning, "the page says %q but this MerchantReturnPolicy does not: set %s", f.said, describeReturnFact(f))
		}
	}
	if link != "" && !slices.ContainsFunc(policies, func(l located) bool {
		return slices.ContainsFunc(strs(l.n, "merchantReturnLink"), func(s string) bool { return normLink(s) == link })
	}) {
		c.add(policies[0].path, Warning, "the page links the return policy (%s): set merchantReturnLink %s on the MerchantReturnPolicy", link, link)
	}
}

// quoteAll is each string quoted, joined with commas.
func quoteAll(ss []string) string {
	q := make([]string, len(ss))
	for i, s := range ss {
		q[i] = strconv.Quote(s)
	}
	return strings.Join(q, ", ")
}

func describeReturnFact(f returnFact) string {
	switch f.prop {
	case "merchantReturnDays":
		return "merchantReturnDays " + f.member
	case "returnPolicySeasonalOverride":
		return "returnPolicySeasonalOverride, a MerchantReturnPolicySeasonalOverride (startDate, endDate, merchantReturnDays) for the dates the page gives"
	}
	return f.prop + " https://schema.org/" + f.member
}

var (
	basedIn     = regexp.MustCompile(`\b(?i:based in\s+(?:the\s+)?)([A-Z][A-Za-z]+(?: [A-Z][A-Za-z]+)?)`)
	openingTime = regexp.MustCompile(`(?i)\b(?:mon(?:day)?|tue(?:s(?:day)?)?|wed(?:nesday)?|thu(?:rs(?:day)?)?|fri(?:day)?|sat(?:urday)?|sun(?:day)?|weekdays|open today)\b[^.\n]{0,40}?\b\d{1,2}(?:[:.]\d{2})?\s*(?:am|pm)?\s*(?:-|–|to)\s*\d{1,2}(?:[:.]\d{2})?\s*(?:am|pm)?`)
	mapSrc      = regexp.MustCompile(`(?i)google\.[a-z.]+/maps|maps\.google\.|openstreetmap\.org|bing\.com/maps|maps\.apple\.com`)
)

// whereAndWhen warns on the business's address, location, opening hours and
// map that the page shows and the markup lacks, and on a contact page whose
// business has no ContactPoint.
func (c *checker) whereAndWhen(all []located, p *pageFacts) {
	hasAddress := hasKey(all, "address") || c.has(all, "PostalAddress")
	if addr := p.address(); addr != "" && !hasAddress {
		c.add(pagePath, Warning, "the page shows the address %q: set the business's address, a PostalAddress (streetAddress, addressLocality, addressRegion, postalCode, addressCountry)", clip(addr, 80))
	} else if m := basedIn.FindStringSubmatch(p.main); m != nil && !hasAddress && (p.kind == kindContact || p.kind == kindAbout) {
		// elsewhere "based in" is about someone else: movers based in Queens
		code := orDefault(countryCodes[strings.ToLower(m[1])], "its two-letter code")
		c.add(pagePath, Warning, "the page says the business is based in %s: set its address, a PostalAddress with addressCountry %s", m[1], code)
	}
	if m := openingTime.FindString(p.main); m != "" && !hasKey(all, "openingHoursSpecification") && !hasKey(all, "openingHours") && !hasKey(all, "hoursAvailable") {
		c.add(pagePath, Warning, "the page gives opening hours (%q): set the store's openingHoursSpecification, an OpeningHoursSpecification for each run of days (dayOfWeek, DayOfWeek members such as https://schema.org/Monday, opens, closes); an online shop's hours go on its ContactPoint as hoursAvailable", m)
	}
	if p.doc != nil && !hasKey(all, "geo") {
		for _, f := range queryAll(p.doc, "iframe[src]") {
			if src, _ := attr(f, "src"); mapSrc.MatchString(src) {
				c.add("html:"+selectorFor(p.doc, f), Warning, "the page embeds a map of the business: set its hasMap (the map's URL) and geo, GeoCoordinates (latitude, longitude)")
				break
			}
		}
	}
	if p.kind == kindContact && !c.has(all, "ContactPoint") {
		c.add(pagePath, Warning, "the page is the business's contact page: give the business a contactPoint, a ContactPoint (contactType customer service, email, telephone, areaServed, availableLanguage)")
	}
}

var (
	paymentNames = regexp.MustCompile(`\b(Visa|Mastercard|MasterCard|American Express|Amex|Diners Club|Discover|JCB|UnionPay|PayPal|Afterpay|Zip Pay|Klarna|Apple Pay|Google Pay|Shop Pay|Laybuy|Humm)\b`)
	cardNames    = regexp.MustCompile(`(?i)^(visa|mastercard|american express|amex|diners club|discover|jcb|unionpay)$`)
	payTypes     = []struct {
		re     *regexp.Regexp
		member string
	}{
		{regexp.MustCompile(`(?i)\b(?:bank transfer|direct deposit)\b`), "ByBankTransferInAdvance"},
		{regexp.MustCompile(`(?i)\bcash on delivery\b`), "COD"},
		{regexp.MustCompile(`(?i)\b(?:take|accept|pay(?:ing)? (?:by|with|in))\s+(?:\w+\s+){0,2}?cash\b|\bcash and cards?\b`), "Cash"},
	}
)

// payments warns on the payment methods the page shows that no node lists as
// an acceptedPaymentMethod.
func (c *checker) payments(all []located, p *pageFacts) {
	if hasKey(all, "acceptedPaymentMethod") {
		return
	}
	var names []string
	add := func(s string) {
		if s != "" && !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, s) }) {
			names = append(names, s)
		}
	}
	for _, m := range paymentNames.FindAllString(p.text, -1) {
		add(m)
	}
	if p.doc != nil {
		for _, n := range queryAll(p.doc, "img[alt], [aria-label], [title]") {
			for _, k := range []string{"alt", "aria-label", "title"} {
				if v, _ := attr(n, k); paymentNames.MatchString(v) && len(strings.Fields(v)) <= 3 {
					add(paymentNames.FindString(v))
				}
			}
		}
	}
	var types []string
	for _, pt := range payTypes {
		if pt.re.MatchString(p.main) {
			types = append(types, "https://schema.org/"+pt.member)
		}
	}
	if len(names) == 0 && len(types) == 0 {
		return
	}
	var cards, others []string
	for _, n := range names {
		if cardNames.MatchString(n) {
			cards = append(cards, n)
		} else {
			others = append(others, n)
		}
	}
	var set []string
	if len(cards) > 0 {
		set = append(set, "a PaymentCard for each card ("+strings.Join(cards, ", ")+")")
	}
	if len(others) > 0 {
		set = append(set, "a PaymentMethod for each other way to pay ("+strings.Join(others, ", ")+")")
	}
	if len(types) > 0 {
		set = append(set, "a PaymentMethod with paymentMethodType "+strings.Join(types, " and ")+" (a PaymentMethodType)")
	}
	c.add(pagePath, Warning, "the page shows how to pay (%s): set acceptedPaymentMethod on the business: %s", strings.Join(append(slices.Clone(names), types...), ", "), strings.Join(set, "; "))
}

// question is a question the page asks and answers, and the heading of the
// FAQ section it is in.
type question struct {
	q, section string
	n          *xhtml.Node
}

var faqHeading = regexp.MustCompile(`(?i)\bfaqs?\b|frequently asked|\bquestions\b|\bq\s?&\s?a\b`)

// questions are the questions the page answers outside its header, footer
// and navs: a details summary, a heading, a dt or an accordion button that
// ends in a question mark and has an answer after it, under an FAQ heading
// (FAQ, Frequently asked questions, Questions) or on an FAQ page. Two or more
// details whose summaries ask questions are an FAQ wherever they are. A
// question heading elsewhere ("How much do movers cost by the hour?") titles
// a section of an article, and a form's step ("Where are you moving?") asks
// the visitor.
func (p *pageFacts) questions() []question {
	if p.doc == nil {
		return nil
	}
	clean := func(q string) string {
		return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(q), "+−-–▾▸›>"))
	}
	asks := func(q string) bool { return strings.HasSuffix(q, "?") && len(strings.Fields(q)) >= 3 }
	var out, summaries []question
	add := func(list *[]question, q, section string, n *xhtml.Node) {
		if q = clean(q); asks(q) && !slices.ContainsFunc(*list, func(o question) bool { return o.q == q }) {
			*list = append(*list, question{q, section, n})
		}
	}
	// the headings over the node, by level; a question heading counts only
	// under an FAQ heading of a higher level
	var heads [7]string
	faq := func(above int) string {
		if p.kind == kindFAQ {
			return orDefault(p.h1(), "FAQ")
		}
		for i := above - 1; i >= 1; i-- {
			if faqHeading.MatchString(heads[i]) {
				return heads[i]
			}
		}
		return ""
	}
	visit(p.doc, func(n *xhtml.Node) {
		if chrome(n) {
			return
		}
		level := 0
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			level = int(n.Data[1] - '0')
		}
		if level > 0 {
			heads[level] = htmlText(n)
			for i := level + 1; i < len(heads); i++ {
				heads[i] = ""
			}
		}
		section := faq(len(heads))
		if level > 0 {
			section = faq(level)
		}
		switch n.DataAtom {
		case atom.Summary:
			add(&summaries, htmlText(n), section, n)
			if section != "" {
				add(&out, htmlText(n), section, n)
			}
		case atom.H2, atom.H3, atom.H4, atom.H5, atom.Dt:
			if next := nextElement(n); section != "" && next != nil && len(strings.Fields(htmlText(next))) >= 3 {
				add(&out, htmlText(n), section, n)
			}
		case atom.Button:
			if v, _ := attr(n, "aria-expanded"); v != "" && section != "" {
				add(&out, htmlText(n), section, n)
			}
		}
	})
	if len(summaries) >= 2 {
		for _, s := range summaries {
			add(&out, s.q, orDefault(s.section, "questions"), s.n)
		}
	}
	return out
}

func nextElement(n *xhtml.Node) *xhtml.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == xhtml.ElementNode {
			return s
		}
	}
	return nil
}

// faq warns on questions the page answers that no Question carries.
func (c *checker) faq(all []located, p *pageFacts) {
	var asked []string
	for _, l := range all {
		if c.is(l.n, "Question") {
			for _, s := range strs(l.n, "name") {
				asked = append(asked, normWords(s))
			}
		}
	}
	qs := p.questions()
	var missing []string
	for _, q := range qs {
		if !slices.Contains(asked, normWords(q.q)) {
			missing = append(missing, q.q)
		}
	}
	if len(missing) == 0 {
		return
	}
	list := missing
	if len(list) > 4 {
		list = append(list[:4:4], "…")
	}
	if !c.has(all, "FAQPage") {
		where := "an FAQPage as a part of the WebPage (@id the page's URL + #faq, url, name, isPartOf the WebPage, in its hasPart)"
		if p.kind == kindFAQ {
			where = "the page itself as an FAQPage"
		}
		c.add(pagePath, Warning, "the page has an FAQ section (%q, %d questions: %s) but no FAQPage: add %s with a Question for each under mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)", clip(qs[0].section, 80), len(missing), strings.Join(list, " "), where)
		return
	}
	c.add(pagePath, Warning, "the page answers %d questions (%s) that no Question carries: add a Question for each to the FAQPage's mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)", len(missing), strings.Join(list, " "))
}
