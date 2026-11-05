package keyword

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(DescriptionDensity{}) }

// DescriptionDensity warns when a description on the page's own nodes uses the
// target keyword, but under the target density: the share of its words the
// keyword fills. A description with no use of the keyword is the
// lacks-keyword check's. It needs a keyword.
type DescriptionDensity struct{}

// ID is the check's ID.
func (DescriptionDensity) ID() string { return "keyword/description-density" }

// Run reports each description whose keyword density is under the target.
func (DescriptionDensity) Run(c *audit.Context) []audit.Issue {
	s := scopeOf(c)
	if s == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range s.owned {
		for _, prop := range descriptionProps {
			for _, v := range jsonld.Strs(l.N, prop) {
				h, w := s.measure(v)
				if d := s.density(h, w); h > 0 && d < s.target {
					out = append(out, audit.Issuef(l.Path+"."+prop, audit.Warning, "%s gives the keyword %q %.1f%% of its words (%d use in %d words), under the %.1f%% target: use it more, or say less", prop, s.keyword, d, h, w, s.target))
				}
			}
		}
	}
	return out
}
