package onpage

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(InvalidSelector{}) }

// InvalidSelector warns about an element node whose cssSelector is not valid
// CSS: it can mark nothing up. It needs the page's HTML, on a page search
// engines index.
type InvalidSelector struct{}

// ID is the check's ID.
func (InvalidSelector) ID() string { return "onpage/invalid-selector" }

// Run reports each cssSelector that is not valid CSS.
func (InvalidSelector) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range analyse(c).invalid {
		out = append(out, audit.Issuef(f.path, audit.Warning, "cssSelector %q is not a valid CSS selector", f.selector))
	}
	return out
}
