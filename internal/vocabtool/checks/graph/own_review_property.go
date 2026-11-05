package graph

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(OwnReviewProperty{}) }

// OwnReviewProperty notes the review and aggregateRating that the page's own
// publisher or provider carries itself: the markup is valid, but Google shows
// no review stars for self-serving reviews.
type OwnReviewProperty struct{}

// ID is the check's ID.
func (OwnReviewProperty) ID() string { return "graph/own-review-property" }

// Run reports each review or rating the page's own business or person carries.
func (OwnReviewProperty) Run(c *audit.Context) []audit.Issue {
	own := ownOf(c)
	var out []audit.Issue
	for _, l := range c.Nodes {
		if own.has(l.N) {
			for _, k := range []string{"review", "aggregateRating"} {
				if _, ok := l.N.Get(k); ok {
					out = append(out, audit.Issuef(l.Path+"."+k, audit.Info, "the page's own business or person carries %s: the markup is valid, but Google shows no review stars for self-serving reviews", k))
				}
			}
		}
	}
	return out
}
