package graph

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(ReviewAuthor{}) }

// ReviewAuthor warns about a Review with no author: a review says who wrote
// it.
type ReviewAuthor struct{}

// ID is the check's ID.
func (ReviewAuthor) ID() string { return "graph/review-author" }

// Run reports each Review with no author.
func (ReviewAuthor) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, l := range reviewsOf(c) {
		if _, ok := l.N.Get("author"); !ok {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "this Review has no author: name who wrote it"))
		}
	}
	return out
}
