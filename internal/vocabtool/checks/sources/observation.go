package sources

import (
	"regexp"
	"slices"
	"strconv"
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
			n := leadFigure(t)
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

// leadFigure is the first number in a sentence, leaving out a year that
// stands alone: 1900 to 2100 after "in", "since" or "by", or opening the
// sentence before a comma ("In 2023, 4.2 million households moved").
func leadFigure(t string) string {
	for _, loc := range firstCount.FindAllStringIndex(t, -1) {
		// the number's pattern takes a comma after it: "2023,"
		if rangeEnd.MatchString(t[:loc[0]]) {
			continue // the 35 of "2025–35"
		}
		n := strings.TrimRight(t[loc[0]:loc[1]], ",")
		loc[1] = loc[0] + len(n)
		if y, err := strconv.Atoi(n); err == nil && len(n) == 4 && y >= 1900 && y <= 2100 &&
			(yearLead.MatchString(t[:loc[0]]) || yearRange.MatchString(t[loc[1]:]) || (sentenceStart.MatchString(t[:loc[0]]) && strings.HasPrefix(t[loc[1]:], ","))) {
			continue
		}
		return t[loc[0]:loc[1]]
	}
	return ""
}

var (
	yearLead = regexp.MustCompile(`(?i)\b(?:in|since|by|from|until|through|between|and|or|to)\s+$`)
	// yearRange follows the first year of a range: "2025–35", "2020-2024".
	yearRange = regexp.MustCompile(`^\s*[–—-]\s*(?:19|20)?\d\d\b`)
	// rangeEnd ends the text before the last year of a range.
	rangeEnd      = regexp.MustCompile(`(?:19|20)\d\d\s*[–—-]\s*$`)
	sentenceStart = regexp.MustCompile(`(?:^|[.!?]\s+)$`)
)

// inside reports an element under one of the tags.
func inside(n *html.Node, tags ...atom.Atom) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if slices.Contains(tags, e.DataAtom) {
			return true
		}
	}
	return false
}
