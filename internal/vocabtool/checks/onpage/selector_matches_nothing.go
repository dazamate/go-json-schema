package onpage

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(SelectorMatchesNothing{}) }

// SelectorMatchesNothing warns about an element node whose cssSelector
// matches nothing on the page: it marks nothing up. It needs the page's HTML,
// on a page search engines index.
type SelectorMatchesNothing struct{}

// ID is the check's ID.
func (SelectorMatchesNothing) ID() string { return "onpage/selector-matches-nothing" }

// Run reports each cssSelector that matches nothing.
func (SelectorMatchesNothing) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range analyse(c).nothing {
		out = append(out, audit.Issuef(f.path, audit.Warning, "cssSelector %q matches nothing on the page", f.selector))
	}
	return out
}
