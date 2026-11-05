// Package sources holds the checks of what a page shows from outside itself:
// a block of statistics with its source is a Dataset, its lead figure an
// Observation, and a table of pay by percentile an Occupation's
// estimatedSalary; the rules, permits and agencies the page cites, the
// documents it links and the sources it credits are each marked up too.
package sources

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	statSource = regexp.MustCompile(`(?i)\b(?:census|survey|statistics|statistical|noaa|climate|normals|bls|oews|acs|population|data)\b`)
	payWords   = regexp.MustCompile(`(?i)\b(?:pay|paid|wages?|earn(?:s|ings)?|salar(?:y|ies))\b`)
	percentile = regexp.MustCompile(`(?i)\b(?:tenth|middle half|median|typical|\d{2}th percentile|quartile)\b`)
	// percentileHead is a column head naming a percentile: "P10", "25th
	// percentile", "Top 10%", "Lower quartile".
	percentileHead = regexp.MustCompile(`(?i)^(?:p(?:10|20|25|75|80|90)|(?:10|20|25|75|80|90)(?:th|%)(?:\s+percentile)?|percentile\s+(?:10|25|75|90)|(?:top|bottom|lowest|highest)\s+(?:10|25)%|(?:lower|upper)\s+quartile)$`)
	moneyCell      = regexp.MustCompile(`[$£€₹]\s?\d`)
	sourceText     = regexp.MustCompile(`\bSources?:\s+(.{3,120})`)
	// sourceStart is a source note: an element's text starts with "Source:".
	sourceStart = regexp.MustCompile(`^\s*Sources?:\s+(.{3,120})`)
	sentenceEnd = regexp.MustCompile(`[a-z0-9)]\.\s+[A-Z]`)
)

// dataBlock is a block of data the page shows: its section, caption and
// source.
type dataBlock struct {
	n               *html.Node
	caption, source string
	byHead          bool // a table whose column heads are percentiles
}

// pay reports a block of pay by percentile: its caption is about pay, and
// its text gives percentiles.
func (b dataBlock) pay() bool {
	return b.byHead || payWords.MatchString(b.caption) && percentile.MatchString(webpage.HTMLText(b.n))
}

// statistical reports a block whose source is a statistical one: a census,
// a survey, a government's statistics.
func (b dataBlock) statistical() bool { return b.source != "" && statSource.MatchString(b.source) }

// dataBlocks are the sections of a page search engines index that show a
// captioned figure or table, each with its source: the first cite in it,
// else its "Source:" line, up to the colon that starts the detail.
func dataBlocks(c *audit.Context) []dataBlock {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	var out []dataBlock
	for _, cap := range webpage.QueryAll(c.Page.Doc, "figcaption, caption") {
		if webpage.Chrome(cap) || !webpage.Shown(cap) {
			continue
		}
		var box *html.Node
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
		} else if line := sourceNote(box); line != "" {
			source = line
		} else if m := sourceText.FindStringSubmatch(webpage.HTMLText(box)); m != nil {
			source = endSentence(m[1])
		}
		source, _, _ = strings.Cut(source, ":")
		out = append(out, dataBlock{n: box, caption: webpage.HTMLText(cap), source: strings.TrimRight(strings.TrimSpace(source), ".")})
	}
	return out
}

// sourceNote is the text after "Source:" in the first element of the block
// that starts with it: a source note of its own, not the sentences around it.
func sourceNote(box *html.Node) string {
	for _, e := range webpage.QueryAll(box, "p, small, li, span, div, td") {
		if m := sourceStart.FindStringSubmatch(webpage.HTMLText(e)); m != nil {
			return endSentence(m[1])
		}
	}
	return ""
}

// endSentence is the text up to the end of its first sentence: a full stop
// after a word or figure, then a capital ("Statistics. Employment"). The dots of
// "U.S." do not end it.
func endSentence(s string) string {
	if loc := sentenceEnd.FindStringIndex(s); loc != nil {
		return s[:loc[0]+2]
	}
	return s
}

// payTables are the pay blocks of the page: the tables dataBlocks finds with
// a pay caption, and the tables whose column heads are percentiles (P10, 25th
// percentile) over amounts of money, with or without a caption or a source.
// The latter is named by the heading above it.
func payTables(c *audit.Context) []dataBlock {
	var out []dataBlock
	for _, b := range dataBlocks(c) {
		if b.pay() {
			out = append(out, b)
		}
	}
	if !c.Indexed() || c.Page.Doc == nil {
		return out
	}
	for _, t := range webpage.QueryAll(c.Page.Doc, "table") {
		if webpage.Chrome(t) || !webpage.Shown(t) || slices.ContainsFunc(out, func(b dataBlock) bool { return slices.Contains(webpage.Ancestors(t), b.n) }) {
			continue
		}
		heads := 0
		for _, th := range webpage.QueryAll(t, "th") {
			if percentileHead.MatchString(strings.TrimSpace(webpage.HTMLText(th))) {
				heads++
			}
		}
		if heads < 2 || !moneyCell.MatchString(webpage.HTMLText(t)) {
			continue
		}
		caption := ""
		if cs := webpage.QueryAll(t, "caption"); len(cs) > 0 {
			caption = webpage.HTMLText(cs[0])
		}
		if caption == "" {
			caption = headingAbove(t)
		}
		out = append(out, dataBlock{n: t, caption: caption, byHead: true})
	}
	return out
}

// headingAbove is the text of the last heading before the element in the
// document.
func headingAbove(n *html.Node) string {
	text := ""
	for e := n; e != nil; e = e.Parent {
		for p := e.PrevSibling; p != nil; p = p.PrevSibling {
			if webpage.HeadingLevel(p) > 0 {
				return webpage.HTMLText(p)
			}
			for _, h := range slices.Backward(webpage.QueryAll(p, "h1, h2, h3, h4, h5, h6")) {
				return webpage.HTMLText(h)
			}
		}
	}
	return text
}
