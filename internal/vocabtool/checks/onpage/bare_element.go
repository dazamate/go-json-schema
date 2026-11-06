package onpage

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(BareElement{}) }

// BareElement warns about an element node with no cssSelector when it does
// not mark up the page's one section of its type: with no selector the node
// cannot say which section it means. It needs the page's HTML, on a page
// search engines index.
type BareElement struct{}

// ID is the check's ID.
func (BareElement) ID() string { return "onpage/bare-element" }

// Run reports each element node that marks up no one section for want of a cssSelector.
func (BareElement) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range analyse(c).bare {
		out = append(out, audit.Issuef(f.path, audit.Warning, "%s has no cssSelector and the page has %d of its sections: give it the cssSelector of the one it marks up", f.ty, f.sections))
	}
	return out
}
