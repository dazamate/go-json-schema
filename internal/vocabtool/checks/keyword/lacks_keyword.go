package keyword

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(LacksKeyword{}) }

// LacksKeyword warns when a text property of the page's own nodes has no use
// of the target keyword: a name, a headline, or a description. A property
// the node lacks altogether is the missing-property check's. It needs a
// keyword.
type LacksKeyword struct{}

// ID is the check's ID.
func (LacksKeyword) ID() string { return "keyword/lacks-keyword" }

// Run reports each name, headline and description without the keyword.
func (LacksKeyword) Run(c *audit.Context) []audit.Issue {
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
				vals := jsonld.Strs(l.N, prop)
				if len(vals) > 0 && !slices.Contains(descriptionProps, prop) && !s.carries(vals) {
					out = append(out, audit.Issuef(l.Path+"."+prop, audit.Warning, "%s %q lacks the keyword %q", prop, txt.Clip(vals[0], 80), s.keyword))
				}
			}
		}
		for _, prop := range descriptionProps {
			for _, v := range jsonld.Strs(l.N, prop) {
				if h, _ := s.measure(v); h == 0 {
					out = append(out, audit.Issuef(l.Path+"."+prop, audit.Warning, "%s %q lacks the keyword %q", prop, txt.Clip(v, 80), s.keyword))
				}
			}
		}
	}
	return out
}
