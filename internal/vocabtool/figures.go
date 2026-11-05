package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

// Figure checks: the charts a page draws, the summary it gives, the vehicles
// a table lists, the terms its copy defines and the quantities its tools
// measure.

// figures runs the figure checks.
func (c *checker) figures(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
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
func (c *checker) charts(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	var first *xhtml.Node
	var caps []string
	for _, f := range webpage.QueryAll(p.Doc, "figure") {
		fc := webpage.QueryAll(f, "figcaption")
		if webpage.Chrome(f) || !webpage.Shown(f) || len(fc) == 0 || len(webpage.QueryAll(f, "svg")) == 0 || len(webpage.QueryAll(f, "img, video, iframe, table")) > 0 {
			continue
		}
		cp := webpage.HTMLText(fc[0])
		if cp == "" || heldText(d, cp) {
			continue
		}
		if first == nil {
			first = f
		}
		caps = append(caps, fmt.Sprintf("%q", txt.Clip(cp, 80)))
	}
	if first == nil {
		return
	}
	what := "the chart " + caps[0]
	if len(caps) > 1 {
		what = fmt.Sprintf("%d charts (%s)", len(caps), txt.Examples(caps))
	}
	c.add("html:"+webpage.SelectorFor(p.Doc, first), Warning, "the page draws %s with no ImageObject: add one for each (caption word for word, encodingFormat image/svg+xml, about the Dataset it draws) as the image of the WebPage or the Dataset", what)
}

// summary warns on a block that sums the page up (Key takeaways, TL;DR, At
// a glance) when the markup has no abstract and no speakable part.
func (c *checker) summary(all []jsonld.Node, p *webpage.Facts) {
	if jsonld.HasKey(all, "abstract") || jsonld.HasKey(all, "speakable") || p.Sells() {
		return
	}
	for _, n := range webpage.QueryAll(p.Doc, "aside, section, div, ul, ol") {
		if webpage.Chrome(n) || !webpage.Shown(n) {
			continue
		}
		label, _ := webpage.Attr(n, "aria-label")
		if !webpage.SummaryLabel.MatchString(label) {
			h := webpage.QueryAll(n, "h2, h3")
			if len(h) == 0 || len(strings.Fields(webpage.HTMLText(h[0]))) > 4 || !webpage.SummaryLabel.MatchString(webpage.HTMLText(h[0])) {
				continue
			}
			label = webpage.HTMLText(h[0])
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page's %q block sums the page up: set its text as the Article's abstract, or point a SpeakableSpecification's cssSelector at it (the WebPage's speakable), or give each line as a Statement (text) in the WebPage's hasPart", label)
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
func (c *checker) vehicles(p *webpage.Facts, d *jsonld.Index) {
	for _, tbl := range webpage.QueryAll(p.Doc, "table") {
		if webpage.Chrome(tbl) || !capacity.MatchString(webpage.HTMLText(tbl)) {
			continue
		}
		head := ""
		for _, n := range webpage.QueryAll(tbl, "caption, thead, tr:first-child th") {
			head += " " + webpage.HTMLText(n)
		}
		every := vehicleName.MatchString(head) || vehicleKind.MatchString(head)
		var names []string
		for _, tr := range webpage.QueryAll(tbl, "tr") {
			cells := webpage.QueryAll(tr, "th, td")
			if len(cells) < 2 || len(webpage.QueryAll(tr, "td")) == 0 {
				continue
			}
			if n := webpage.HTMLText(cells[0]); (every || vehicleName.MatchString(n)) && len(strings.Fields(n)) <= 5 && !heldText(d, n) {
				names = append(names, n)
			}
		}
		if len(names) < 2 {
			continue
		}
		name := ""
		if cs := webpage.QueryAll(tbl, "caption"); len(cs) > 0 {
			name = webpage.HTMLText(cs[0])
		}
		at := tbl
		if b := idBox(tbl); b != nil && b.DataAtom != 0 && slices.Contains([]string{"section", "article", "div", "figure"}, b.Data) {
			at = b
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, at), Warning, "the page's table %q lists vehicles (%s) with their capacity: add a Vehicle for each (name, cargoVolume, payload, vehicleConfiguration), in an ItemList or as the items of the Offer or Service they are for. A Vehicle is a kind of Product, and Search Console's Product snippets report marks one without offers, review or aggregateRating invalid: give each offers with the rental or sale price the page shows for it (an AggregateOffer for a range: lowPrice, highPrice, priceCurrency, offerCount)", name, txt.Examples(names))
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
func (c *checker) definitions(all []jsonld.Node, p *webpage.Facts) {
	var named []string
	for _, l := range all {
		if c.is(l.N, "DefinedTerm") {
			for _, s := range jsonld.Strs(l.N, "name") {
				named = append(named, txt.NormWords(s))
			}
		}
	}
	lower := strings.ToLower(p.Main)
	var terms []string
	for _, re := range []*regexp.Regexp{appositive, meansDef} {
		for _, m := range re.FindAllStringSubmatch(p.Main, -1) {
			t := m[1]
			if strings.Count(lower, strings.ToLower(t)) < 3 || slices.Contains(named, txt.NormWords(t)) || slices.Contains(terms, t) {
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
	c.add(pagePath, Warning, "the page defines %s but no DefinedTerm names it: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, description from the page's definition)", txt.Examples(quoted))
}

var quantity = regexp.MustCompile(`(?i)(?:^|[^$\d.,])(\d[\d,]*(?:\.\d+)?(?:\s*(?:–|-|to)\s*\d[\d,]*(?:\.\d+)?)?)\s*(hours?|hrs?|minutes?|mins?|days?|weeks?|months?|years?|miles?|mi|km|lbs?|pounds?|kg|cu\.? ?ft|cubic feet|sq\.? ?ft|square feet|ft|feet|°F|°C)\b`)

// quantities warn on measured quantities a tool's output or a data element
// gives (4–6.5 hours, 4,500–6,500 lb) whose numbers the markup does not hold.
// A shop's sizes are the product checks'.
func (c *checker) quantities(all []jsonld.Node, p *webpage.Facts) {
	if p.Sells() {
		return
	}
	held := map[float64]bool{}
	for _, l := range all {
		for _, k := range l.N.Keys {
			for _, s := range jsonld.Scalars(l.N, k) {
				for _, n := range numberIn.FindAllString(strings.ReplaceAll(s, ",", ""), -1) {
					if f, ok := txt.Amount(n); ok {
						held[f] = true
					}
				}
			}
		}
	}
	var list []string
	for _, n := range webpage.QueryAll(p.Doc, "output, data") {
		if webpage.Chrome(n) {
			continue
		}
		for _, m := range quantity.FindAllStringSubmatch(webpage.HTMLText(n), -1) {
			q := strings.TrimSpace(m[1] + " " + m[2])
			all := true
			for _, s := range numberIn.FindAllString(strings.ReplaceAll(m[1], ",", ""), -1) {
				if f, _ := txt.Amount(s); !held[f] {
					all = false
				}
			}
			if !all && !slices.Contains(list, q) {
				list = append(list, q)
			}
		}
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page's tools and figures give quantities (%s): mark each up as a QuantitativeValue (value, or minValue and maxValue, unitCode such as HUR, DAY, MON, SMI, LBR or FTQ) on the thing it measures; a property whose range is a Duration, Distance or Mass takes it as that type's text (PT4H, \"10 mi\", \"6500 lb\")", txt.Examples(list))
	}
}

var proCon = regexp.MustCompile(`(?i)^(?:for it|against it|pros|cons|advantages|disadvantages|upsides?|downsides?)$`)

// options warn on a section that weighs options against each other, two or
// more cards each with pros and cons headings, when no Guide carries it.
func (c *checker) options(all []jsonld.Node, p *webpage.Facts) {
	if c.has(all, "Guide") {
		return
	}
	for _, s := range webpage.QueryAll(p.Doc, "section") {
		if webpage.Chrome(s) || !webpage.Shown(s) {
			continue
		}
		var names []string
		for _, a := range webpage.QueryAll(s, "article, li") {
			hs := webpage.QueryAll(a, "h3, h4, h5")
			pc := 0
			for _, h := range hs {
				if proCon.MatchString(webpage.HTMLText(h)) {
					pc++
				}
			}
			if pc >= 2 && !proCon.MatchString(webpage.HTMLText(hs[0])) {
				names = append(names, webpage.HTMLText(hs[0]))
			}
		}
		if len(names) >= 2 {
			c.add("html:"+webpage.SelectorFor(p.Doc, s), Warning, "the page weighs options (%s) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)", txt.Examples(names))
			return
		}
	}
}
