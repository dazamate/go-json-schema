package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	xhtml "golang.org/x/net/html"
)

// Figure checks: the charts a page draws, the summary it gives, the vehicles
// a table lists, the terms its copy defines and the quantities its tools
// measure.

// figures runs the figure checks.
func (c *checker) figures(all []located, p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	c.charts(all, p, d)
	c.summary(all, p)
	c.vehicles(p, d)
	c.definitions(all, p)
	c.quantities(all, p)
	c.options(all, p)
}

// charts warn on figures drawn in SVG, with a caption, that no ImageObject
// carries: images reads only img elements.
func (c *checker) charts(all []located, p *pageFacts, d *draftIndex) {
	var first *xhtml.Node
	var caps []string
	for _, f := range queryAll(p.doc, "figure") {
		fc := queryAll(f, "figcaption")
		if chrome(f) || !shown(f) || len(fc) == 0 || len(queryAll(f, "svg")) == 0 || len(queryAll(f, "img, video, iframe, table")) > 0 {
			continue
		}
		cp := htmlText(fc[0])
		if cp == "" || heldText(d, cp) {
			continue
		}
		if first == nil {
			first = f
		}
		caps = append(caps, fmt.Sprintf("%q", clip(cp, 80)))
	}
	if first == nil {
		return
	}
	what := "the chart " + caps[0]
	if len(caps) > 1 {
		what = fmt.Sprintf("%d charts (%s)", len(caps), examples(caps))
	}
	c.add("html:"+selectorFor(p.doc, first), Warning, "the page draws %s with no ImageObject: add one for each (caption word for word, encodingFormat image/svg+xml, about the Dataset it draws) as the image of the WebPage or the Dataset", what)
}

// summary warns on a block that sums the page up (Key takeaways, TL;DR, At
// a glance) when the markup has no abstract and no speakable part.
func (c *checker) summary(all []located, p *pageFacts) {
	if hasKey(all, "abstract") || hasKey(all, "speakable") || p.sells() {
		return
	}
	for _, n := range queryAll(p.doc, "aside, section, div, ul, ol") {
		if chrome(n) || !shown(n) {
			continue
		}
		label, _ := attr(n, "aria-label")
		if !summaryLabel.MatchString(label) {
			h := queryAll(n, "h2, h3")
			if len(h) == 0 || len(strings.Fields(htmlText(h[0]))) > 4 || !summaryLabel.MatchString(htmlText(h[0])) {
				continue
			}
			label = htmlText(h[0])
		}
		c.add("html:"+selectorFor(p.doc, n), Warning, "the page's %q block sums the page up: set its text as the Article's abstract, or point a SpeakableSpecification's cssSelector at it (the WebPage's speakable), or give each line as a Statement (text) in the WebPage's hasPart", label)
		return
	}
}

var (
	vehicleKind = regexp.MustCompile(`(?i)\b(?:trucks|vans|vehicles)\b`)
	vehicleName = regexp.MustCompile(`(?i)\b(?:van|truck|trailer|pickup|lorry|ute|box truck|cube)\b`)
	capacity    = regexp.MustCompile(`(?i)\bcu\.? ?ft\b|\bcubic\b|\bpayload\b|\brated load\b|\bcapacity\b|\bholds?\b|\bspace\b|\blb\b|\bkg\b`)
)

// vehicles warn on a table that lists vehicles with their capacity (truck
// sizes and what each holds) when no node names them. A table whose caption
// or header names vehicles lists one in each row (10 ft, 15 ft).
func (c *checker) vehicles(p *pageFacts, d *draftIndex) {
	for _, tbl := range queryAll(p.doc, "table") {
		if chrome(tbl) || !capacity.MatchString(htmlText(tbl)) {
			continue
		}
		head := ""
		for _, n := range queryAll(tbl, "caption, thead, tr:first-child th") {
			head += " " + htmlText(n)
		}
		every := vehicleName.MatchString(head) || vehicleKind.MatchString(head)
		var names []string
		for _, tr := range queryAll(tbl, "tr") {
			cells := queryAll(tr, "th, td")
			if len(cells) < 2 || len(queryAll(tr, "td")) == 0 {
				continue
			}
			if n := htmlText(cells[0]); (every || vehicleName.MatchString(n)) && len(strings.Fields(n)) <= 5 && !heldText(d, n) {
				names = append(names, n)
			}
		}
		if len(names) < 2 {
			continue
		}
		name := ""
		if cs := queryAll(tbl, "caption"); len(cs) > 0 {
			name = htmlText(cs[0])
		}
		at := tbl
		if b := idBox(tbl); b != nil && b.DataAtom != 0 && slices.Contains([]string{"section", "article", "div", "figure"}, b.Data) {
			at = b
		}
		c.add("html:"+selectorFor(p.doc, at), Warning, "the page's table %q lists vehicles (%s) with their capacity: add a Vehicle for each (name, cargoVolume, payload, vehicleConfiguration), in an ItemList or as the items of the Offer or Service they are for. A Vehicle is a kind of Product, and Search Console's Product snippets report marks one without offers, review or aggregateRating invalid: give each offers with the rental or sale price the page shows for it (an AggregateOffer for a range: lowPrice, highPrice, priceCurrency, offerCount)", name, examples(names))
		return
	}
}

var (
	appositive = regexp.MustCompile(`(?:^|[.!?]\s+)([A-Z][a-z]+(?:[ -][a-z]+){1,3}), (?:the|a|an) [^,.]{8,90}, (?:is|means|pays|covers|gives|lets|protects)\b`)
	meansDef   = regexp.MustCompile(`(?:^|[.!?]\s+)(?:An? |The )?([A-Z][a-z]+(?:[ -][a-z]+){1,3}) (?:means|refers to|is defined as|is the term for)\b`)
)

// definitions warn on terms the copy defines ("Released value, the basic
// cover a mover includes, pays …", "A binding estimate means …") and uses
// again, when no DefinedTerm names them.
func (c *checker) definitions(all []located, p *pageFacts) {
	var named []string
	for _, l := range all {
		if c.is(l.n, "DefinedTerm") {
			for _, s := range strs(l.n, "name") {
				named = append(named, normWords(s))
			}
		}
	}
	lower := strings.ToLower(p.main)
	var terms []string
	for _, re := range []*regexp.Regexp{appositive, meansDef} {
		for _, m := range re.FindAllStringSubmatch(p.main, -1) {
			t := m[1]
			if strings.Count(lower, strings.ToLower(t)) < 3 || slices.Contains(named, normWords(t)) || slices.Contains(terms, t) {
				continue
			}
			terms = append(terms, t)
		}
	}
	if len(terms) == 0 {
		return
	}
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = fmt.Sprintf("%q", t)
	}
	c.add(pagePath, Warning, "the page defines %s but no DefinedTerm names it: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, description from the page's definition)", examples(quoted))
}

var quantity = regexp.MustCompile(`(?i)(?:^|[^$\d.,])(\d[\d,]*(?:\.\d+)?(?:\s*(?:–|-|to)\s*\d[\d,]*(?:\.\d+)?)?)\s*(hours?|hrs?|minutes?|mins?|days?|weeks?|months?|years?|miles?|mi|km|lbs?|pounds?|kg|cu\.? ?ft|cubic feet|sq\.? ?ft|square feet|ft|feet|°F|°C)\b`)

// quantities warn on measured quantities a tool's output or a data element
// gives (4–6.5 hours, 4,500–6,500 lb) whose numbers the markup does not hold.
// A shop's sizes are the product checks'.
func (c *checker) quantities(all []located, p *pageFacts) {
	if p.sells() {
		return
	}
	held := map[float64]bool{}
	for _, l := range all {
		for _, k := range l.n.keys {
			for _, s := range scalars(l.n, k) {
				for _, n := range numberIn.FindAllString(strings.ReplaceAll(s, ",", ""), -1) {
					if f, ok := amount(n); ok {
						held[f] = true
					}
				}
			}
		}
	}
	var list []string
	for _, n := range queryAll(p.doc, "output, data") {
		if chrome(n) {
			continue
		}
		for _, m := range quantity.FindAllStringSubmatch(htmlText(n), -1) {
			q := strings.TrimSpace(m[1] + " " + m[2])
			all := true
			for _, s := range numberIn.FindAllString(strings.ReplaceAll(m[1], ",", ""), -1) {
				if f, _ := amount(s); !held[f] {
					all = false
				}
			}
			if !all && !slices.Contains(list, q) {
				list = append(list, q)
			}
		}
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page's tools and figures give quantities (%s): mark each up as a QuantitativeValue (value, or minValue and maxValue, unitCode such as HUR, DAY, MON, SMI, LBR or FTQ) on the thing it measures; a property whose range is a Duration, Distance or Mass takes it as that type's text (PT4H, \"10 mi\", \"6500 lb\")", examples(list))
	}
}

var proCon = regexp.MustCompile(`(?i)^(?:for it|against it|pros|cons|advantages|disadvantages|upsides?|downsides?)$`)

// options warn on a section that weighs options against each other, two or
// more cards each with pros and cons headings, when no Guide carries it.
func (c *checker) options(all []located, p *pageFacts) {
	if c.has(all, "Guide") {
		return
	}
	for _, s := range queryAll(p.doc, "section") {
		if chrome(s) || !shown(s) {
			continue
		}
		var names []string
		for _, a := range queryAll(s, "article, li") {
			hs := queryAll(a, "h3, h4, h5")
			pc := 0
			for _, h := range hs {
				if proCon.MatchString(htmlText(h)) {
					pc++
				}
			}
			if pc >= 2 && !proCon.MatchString(htmlText(hs[0])) {
				names = append(names, htmlText(hs[0]))
			}
		}
		if len(names) >= 2 {
			c.add("html:"+selectorFor(p.doc, s), Warning, "the page weighs options (%s) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)", examples(names))
			return
		}
	}
}
