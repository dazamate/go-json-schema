// Package sources holds the data checks: a block of statistics with its
// source is a Dataset, its lead figure an Observation, and a table of pay by
// percentile an Occupation's estimatedSalary.
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
	sourceText = regexp.MustCompile(`\bSources?:\s+(.{3,120})`)
)

// dataBlock is a block of data the page shows: its section, caption and
// source.
type dataBlock struct {
	n               *html.Node
	caption, source string
}

// pay reports a block of pay by percentile: its caption is about pay, and
// its text gives percentiles.
func (b dataBlock) pay() bool {
	return payWords.MatchString(b.caption) && percentile.MatchString(webpage.HTMLText(b.n))
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
		} else if m := sourceText.FindStringSubmatch(webpage.HTMLText(box)); m != nil {
			source = m[1]
		}
		source, _, _ = strings.Cut(source, ":")
		out = append(out, dataBlock{box, webpage.HTMLText(cap), strings.TrimSpace(source)})
	}
	return out
}
