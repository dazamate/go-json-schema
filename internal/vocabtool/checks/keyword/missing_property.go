package keyword

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(MissingProperty{}) }

// MissingProperty warns when a node of the page lacks a text property that
// should carry the target keyword: the page's name, alternateName and
// descriptions, an Article's headline and alternativeHeadline, a Service's
// descriptions. It needs a keyword.
type MissingProperty struct{}

// ID is the check's ID.
func (MissingProperty) ID() string { return "keyword/missing-property" }

// Run reports each property the page's own nodes should have and lack.
func (MissingProperty) Run(c *audit.Context) []audit.Issue {
	s := scopeOf(c)
	if s == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range s.owned {
		for _, kp := range keywordProps {
			if !s.takes(c, l, kp.of) {
				continue
			}
			for _, prop := range kp.props {
				if len(jsonld.Strs(l.N, prop)) == 0 {
					out = append(out, audit.Issuef(l.Path, audit.Warning, "no %s: add one that carries the keyword %q", prop, s.keyword))
				}
			}
		}
	}
	return out
}

// takes reports whether the node is of the type and, for a WebPage, is the
// page's own: a WebPage nested in another is not the page.
func (s *scope) takes(c *audit.Context, l jsonld.Node, ty string) bool {
	return c.Is(l.N, ty) && (ty != "WebPage" || s.hasPage && l.N == s.page.N)
}
