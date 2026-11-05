package keyword

import (
	"net/url"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(MarkupDensity{}) }

// MarkupDensity warns when the target keyword fills less than the target
// share of the words of the markup as a whole. Every text value counts, bar
// the @ keywords and the URLs. It needs a keyword.
type MarkupDensity struct{}

// ID is the check's ID.
func (MarkupDensity) ID() string { return "keyword/markup-density" }

// Run reports the draft when the keyword's density across its text is under
// the target.
func (MarkupDensity) Run(c *audit.Context) []audit.Issue {
	s := scopeOf(c)
	if s == nil {
		return nil
	}
	hits, words := 0, 0
	var count func(v any)
	count = func(v any) {
		switch x := v.(type) {
		case *jsonld.Object:
			for _, k := range x.Keys {
				if !strings.HasPrefix(k, "@") || k == "@graph" {
					count(x.Vals[k])
				}
			}
		case []any:
			for _, item := range x {
				count(item)
			}
		case string:
			if u, err := url.Parse(x); err == nil && u.IsAbs() && !strings.Contains(x, " ") {
				return // a URL is not text
			}
			h, w := s.measure(x)
			hits, words = hits+h, words+w
		}
	}
	count(c.Root)
	d := s.density(hits, words)
	if d >= s.target {
		return nil
	}
	return []audit.Issue{audit.Issuef("$", audit.Warning, "the keyword %q fills %.1f%% of the markup's words (%d uses in %d words), under the %.1f%% target: work it into the page's names, headlines and descriptions", s.keyword, d, hits, words, s.target)}
}
