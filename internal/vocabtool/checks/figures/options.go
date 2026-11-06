package figures

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Options{}) }

var proCon = regexp.MustCompile(`(?i)^(?:for it|against it|pros|cons|advantages|disadvantages|upsides?|downsides?)$`)

// Options warns on a section that weighs options against each other, two or
// more cards each with pros and cons headings, when no Guide carries it. It
// needs the page's document, on a page search engines index.
type Options struct{}

// ID is the check's ID.
func (Options) ID() string { return "figures/options" }

// Run reports the page's first such section.
func (Options) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || c.Has("Guide") {
		return nil
	}
	p := c.Page
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
			return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, s), audit.Warning, "the page weighs options (%s) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)", txt.Examples(names))}
		}
	}
	return nil
}
