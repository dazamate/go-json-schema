package vocabtool

import (
	"regexp"
	"slices"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Data checks: a block of statistics with its source is a Dataset, its lead
// figure an Observation, and a table of pay by percentile an Occupation's
// estimatedSalary.

var (
	statSource = regexp.MustCompile(`(?i)\b(?:census|survey|statistics|statistical|noaa|climate|normals|bls|oews|acs|population|data)\b`)
	payWords   = regexp.MustCompile(`(?i)\b(?:pay|paid|wages?|earn(?:s|ings)?|salar(?:y|ies))\b`)
	percentile = regexp.MustCompile(`(?i)\b(?:tenth|middle half|median|typical|\d{2}th percentile|quartile)\b`)
	payFor     = regexp.MustCompile(`(?i)\bpay (?:for|of) ([^,(]+)`)
	firstCount = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?`)
	bedrooms   = regexp.MustCompile(`(?i)\bbedrooms?\b|\brooms\b`)
	sourceText = regexp.MustCompile(`\bSources?:\s+(.{3,120})`)
)

// dataBlock is a block of data the page shows: its section, caption and
// source.
type dataBlock struct {
	n               *xhtml.Node
	caption, source string
}

// dataBlocks are the sections that show a captioned figure or table, each
// with its source: the first cite in it, else its "Source:" line, up to the
// colon that starts the detail.
func (p *pageFacts) dataBlocks() []dataBlock {
	if p.doc == nil {
		return nil
	}
	var out []dataBlock
	for _, cap := range queryAll(p.doc, "figcaption, caption") {
		if chrome(cap) || !shown(cap) {
			continue
		}
		var box *xhtml.Node
		for e := cap.Parent; e != nil && e.DataAtom != atom.Body && e.DataAtom != atom.Main; e = e.Parent {
			if e.DataAtom == atom.Section || e.DataAtom == atom.Article {
				box = e
				break
			}
		}
		if box == nil || slices.ContainsFunc(out, func(b dataBlock) bool { return b.n == box }) {
			continue
		}
		source := ""
		if cs := queryAll(box, "cite"); len(cs) > 0 {
			source = htmlText(cs[0])
		} else if m := sourceText.FindStringSubmatch(htmlText(box)); m != nil {
			source = m[1]
		}
		source, _, _ = strings.Cut(source, ":")
		out = append(out, dataBlock{box, htmlText(cap), strings.TrimSpace(source)})
	}
	return out
}

// data warns on the page's data blocks that no Dataset describes, their lead
// figures that no Observation carries, and a pay table that no Occupation's
// estimatedSalary carries. A pay table's amounts are wages, not prices.
func (c *checker) data(all []located, p *pageFacts, d *draftIndex) {
	var observed []string
	for _, l := range all {
		if c.is(l.n, "Observation") {
			observed = append(observed, scalars(l.n, "value")...)
		}
	}
	for _, b := range p.dataBlocks() {
		text := htmlText(b.n)
		at := "html:" + selectorFor(p.doc, b.n)
		pay := payWords.MatchString(b.caption) && percentile.MatchString(text)
		if pay {
			p.claim(text)
			if !hasKey(all, "estimatedSalary") {
				job := "the occupation"
				if m := payFor.FindStringSubmatch(b.caption); m != nil {
					job = strings.TrimSpace(m[1])
				}
				area := "the area"
				for _, tr := range queryAll(b.n, "tr") {
					if ths := queryAll(tr, "th"); len(ths) > 0 && len(queryAll(tr, "td")) > 0 {
						area = htmlText(ths[0])
						break
					}
				}
				c.add(at, Warning, "the page shows pay percentiles for %q in %s: add an Occupation (name, occupationLocation) with estimatedSalary as a MonetaryAmountDistribution (percentile10, percentile25, median, percentile75, percentile90, currency, unitText HOUR or YEAR)", job, area)
			}
		}
		if b.source == "" || !statSource.MatchString(b.source) {
			continue
		}
		if !heldText(d, b.caption) {
			c.add(at, Warning, "the page shows data from %s (%q) but no Dataset describes it: add one (name and description from the caption and its note, creator or publisher the source, isBasedOn its URL, temporalCoverage from the years, spatialCoverage from the places, variableMeasured from the column headers)", clip(b.source, 100), clip(b.caption, 80))
		}
		if pay {
			continue
		}
		// the block's lead sentence gives its headline figure
		for _, para := range queryAll(b.n, "p") {
			t := htmlText(para)
			// a price is an estimate, not the data's figure
			if inside(para, atom.Figure, atom.Details, atom.Small) || strings.Contains(t, "Source") || len(strings.Fields(t)) > 30 || priceAmount.MatchString(t) {
				continue
			}
			n := firstCount.FindString(t)
			if n == "" {
				continue
			}
			if !slices.Contains(observed, strings.ReplaceAll(n, ",", "")) {
				home := ""
				if bedrooms.MatchString(t) {
					home = "; a FloorPlan (numberOfBedrooms, numberOfRooms) describes the home it counts"
				}
				c.add(at, Warning, "the page states %q from %s: add an Observation (observationAbout the place, variableMeasured a StatisticalVariable with measuredProperty and statType, value, observationDate) to the Dataset%s", t, clip(b.source, 80), home)
			}
			break
		}
	}
}

// inside reports an element under one of the tags.
func inside(n *xhtml.Node, tags ...atom.Atom) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if slices.Contains(tags, e.DataAtom) {
			return true
		}
	}
	return false
}
