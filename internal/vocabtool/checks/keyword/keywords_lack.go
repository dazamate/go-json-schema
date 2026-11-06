package keyword

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(KeywordsLack{}) }

// KeywordsLack warns when the page's keywords property has no use of the
// target keyword. The page's keywords lead with the one it targets. It needs a
// keyword.
type KeywordsLack struct{}

// ID is the check's ID.
func (KeywordsLack) ID() string { return "keyword/keywords-lack" }

// Run reports the page's WebPage when its keywords lack the target keyword.
func (KeywordsLack) Run(c *audit.Context) []audit.Issue {
	s := scopeOf(c)
	if s == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range s.owned {
		if s.hasPage && l.N == s.page.N && !s.carries(jsonld.Strs(l.N, "keywords")) {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "keywords lacks the target keyword %q: put it first", s.keyword))
		}
	}
	return out
}
