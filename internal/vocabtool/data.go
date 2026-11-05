package vocabtool

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
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
func dataBlocks(p *webpage.Facts) []dataBlock {
	if p.Doc == nil {
		return nil
	}
	var out []dataBlock
	for _, cap := range webpage.QueryAll(p.Doc, "figcaption, caption") {
		if webpage.Chrome(cap) || !webpage.Shown(cap) {
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
		if cs := webpage.QueryAll(box, "cite"); len(cs) > 0 {
			source = webpage.HTMLText(cs[0])
		} else if m := sourceText.FindStringSubmatch(webpage.HTMLText(box)); m != nil {
			source = m[1]
		}
		source, _, _ = strings.Cut(source, ":")
		out = append(out, dataBlock{box, webpage.HTMLText(cap), strings.TrimSpace(source)})
	}
	return out
}

// data warns on the page's data blocks that no Dataset describes, their lead
// figures that no Observation carries, and a pay table that no Occupation's
// estimatedSalary carries. A pay table's amounts are wages, not prices.
func (c *checker) data(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	var observed, stated []string
	for _, l := range all {
		if c.is(l.N, "Observation") {
			observed = append(observed, jsonld.Scalars(l.N, "value")...)
			// An Observation named by the sentence carries it: "1 in 6 people
			// moved" leads with 1, which is no figure of its own.
			for _, n := range jsonld.Scalars(l.N, "name") {
				stated = append(stated, strings.Join(strings.Fields(n), " "))
			}
		}
	}
	for _, b := range dataBlocks(p) {
		text := webpage.HTMLText(b.n)
		at := "html:" + webpage.SelectorFor(p.Doc, b.n)
		pay := payWords.MatchString(b.caption) && percentile.MatchString(text)
		if pay {
			c.claim(text)
			if !jsonld.HasKey(all, "estimatedSalary") {
				job := "the occupation"
				if m := payFor.FindStringSubmatch(b.caption); m != nil {
					job = strings.TrimSpace(m[1])
				}
				area := "the area"
				for _, tr := range webpage.QueryAll(b.n, "tr") {
					if ths := webpage.QueryAll(tr, "th"); len(ths) > 0 && len(webpage.QueryAll(tr, "td")) > 0 {
						area = webpage.HTMLText(ths[0])
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
			c.add(at, Warning, "the page shows data from %s (%q) but no Dataset describes it: add one (name and description from the caption and its note, creator or publisher the source, isBasedOn its URL, temporalCoverage from the years, spatialCoverage from the places, variableMeasured from the column headers)", txt.Clip(b.source, 100), txt.Clip(b.caption, 80))
		}
		if pay {
			continue
		}
		// the block's lead sentence gives its headline figure
		for _, para := range webpage.QueryAll(b.n, "p") {
			t := webpage.HTMLText(para)
			// a price is an estimate, not the data's figure
			if inside(para, atom.Figure, atom.Details, atom.Small) || strings.Contains(t, "Source") || len(strings.Fields(t)) > 30 || txt.PriceAmount.MatchString(t) {
				continue
			}
			n := firstCount.FindString(t)
			if n == "" {
				continue
			}
			if !slices.Contains(observed, strings.ReplaceAll(n, ",", "")) && !slices.Contains(stated, strings.Join(strings.Fields(t), " ")) {
				home := ""
				if bedrooms.MatchString(t) {
					home = "; a FloorPlan (numberOfBedrooms, numberOfRooms) describes the home it counts"
				}
				c.add(at, Warning, "the page states %q from %s: add an Observation (observationAbout the place, variableMeasured a StatisticalVariable with measuredProperty and statType, value, observationDate) to the Dataset%s", t, txt.Clip(b.source, 80), home)
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
