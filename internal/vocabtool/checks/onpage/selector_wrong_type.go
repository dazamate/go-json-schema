package onpage

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(SelectorWrongType{}) }

// SelectorWrongType warns about an element node whose cssSelector selects
// only sections of another type: a WPSideBar that selects the site's nav marks
// up the wrong section. It needs the page's HTML, on a page search engines
// index.
type SelectorWrongType struct{}

// ID is the check's ID.
func (SelectorWrongType) ID() string { return "onpage/selector-wrong-type" }

// Run reports each cssSelector that selects only sections of another type.
func (SelectorWrongType) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range analyse(c).wrongType {
		out = append(out, audit.Issuef(f.path, audit.Warning, "cssSelector %q selects a %s, not a %s", f.selector, f.selected, f.want))
	}
	return out
}
