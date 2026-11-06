package graph

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(OwnReview{}) }

// OwnReview notes a Review or AggregateRating whose itemReviewed is the
// page's own publisher or provider: the markup is valid, but Google shows no
// review stars for self-serving reviews.
type OwnReview struct{}

// ID is the check's ID.
func (OwnReview) ID() string { return "graph/own-review" }

// Run reports each review of the page's own business or person.
func (OwnReview) Run(c *audit.Context) []audit.Issue {
	own := ownIDs(c)
	isOwn := func(v any) bool {
		if o, ok := v.(*jsonld.Object); ok {
			return slices.ContainsFunc(jsonld.Strs(o, "@id"), func(id string) bool { return own[id] })
		}
		s, _ := v.(string)
		return own[s]
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		if c.Is(l.N, "Review") || c.Is(l.N, "AggregateRating") {
			raw, _ := l.N.Get("itemReviewed")
			if slices.ContainsFunc(jsonld.Values(raw), isOwn) {
				out = append(out, audit.Issuef(l.Path, audit.Info, "a %s of the page's own business or person: the markup is valid, but Google shows no review stars for self-serving reviews", jsonld.TypeNames(l.N)[0]))
			}
		}
	}
	return out
}
