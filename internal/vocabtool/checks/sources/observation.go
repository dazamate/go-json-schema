package sources

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Observation{}) }

var (
	firstCount = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?`)
	bedrooms   = regexp.MustCompile(`(?i)\bbedrooms?\b|\brooms\b`)
)

// Observation warns on the lead figure of a block of statistics that no
// Observation carries: the block's lead sentence gives its headline figure.
// A pay table's figures are its Occupation's, and a price is an estimate,
// not the data's figure.
type Observation struct{}

// ID is the check's ID.
func (Observation) ID() string { return "sources/observation" }

// Run reports each block's first short paragraph with a figure, unless an
// Observation has the figure as its value or the sentence as its name.
func (Observation) Run(c *audit.Context) []audit.Issue {
	var observed, stated []string
	for _, l := range c.Nodes {
		if c.Is(l.N, "Observation") {
			observed = append(observed, jsonld.Scalars(l.N, "value")...)
			// An Observation named by the sentence carries it: "1 in 6 people
			// moved" leads with 1, which is no figure of its own.
			for _, n := range jsonld.Scalars(l.N, "name") {
				stated = append(stated, strings.Join(strings.Fields(n), " "))
			}
		}
	}
	var out []audit.Issue
	for _, b := range dataBlocks(c) {
		if !b.statistical() || b.pay() {
			continue
		}
		for _, para := range webpage.QueryAll(b.n, "p") {
			t := webpage.HTMLText(para)
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
				out = append(out, audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, b.n), audit.Warning, "the page states %q from %s: add an Observation (observationAbout the place, variableMeasured a StatisticalVariable with measuredProperty and statType, value, observationDate) to the Dataset%s", t, txt.Clip(b.source, 80), home))
			}
			break
		}
	}
	return out
}

// inside reports an element under one of the tags.
func inside(n *html.Node, tags ...atom.Atom) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if slices.Contains(tags, e.DataAtom) {
			return true
		}
	}
	return false
}
