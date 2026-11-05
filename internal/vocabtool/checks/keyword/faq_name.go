package keyword

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(FAQName{}) }

// FAQName warns when the name of an FAQPage inside the page lacks the target
// keyword. The FAQ is a section of the page with a heading of its own, and the
// heading is named for the keyword. An FAQPage that is the page itself is held
// to the page's properties instead. It needs a keyword.
type FAQName struct{}

// ID is the check's ID.
func (FAQName) ID() string { return "keyword/faq-name" }

// Run reports each FAQ whose name has no use of the keyword.
func (FAQName) Run(c *audit.Context) []audit.Issue {
	s := scopeOf(c)
	if s == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range s.owned {
		if c.Is(l.N, "FAQPage") && (!s.hasPage || l.N != s.page.N) && !s.carries(jsonld.Strs(l.N, "name")) {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "the FAQ's name lacks the keyword %q: name it for the keyword, as its heading on the page", s.keyword))
		}
	}
	return out
}
