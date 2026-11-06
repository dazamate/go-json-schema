package onpage

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(UnmarkedSection{}) }

// UnmarkedSection warns about a section of the page (its header, footer,
// nav, sidebar, table or ad slot) that no element node marks up, and says
// which node to add. It needs the page's HTML, on a page search engines
// index.
type UnmarkedSection struct{}

// ID is the check's ID.
func (UnmarkedSection) ID() string { return "onpage/unmarked-section" }

// Run reports each section no element node marks up.
func (UnmarkedSection) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range analyse(c).unmarked {
		out = append(out, audit.Issuef("html:"+f.selector, audit.Warning, `the page's <%s> has no %s: add {"@type": %q, "cssSelector": %q} to the WebPage's hasPart`, f.tag, f.ty, f.ty, f.selector))
	}
	return out
}
