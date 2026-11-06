package graph

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ReviewItem{}) }

// ReviewItem warns about a Review with no itemReviewed, unless it is nested
// in the review property of the item it reviews: that item is then what it
// reviews.
type ReviewItem struct{}

// ID is the check's ID.
func (ReviewItem) ID() string { return "graph/review-item" }

// reviewNested matches the path of a node held in a review property.
var reviewNested = regexp.MustCompile(`\.review(?:\[\d+\])?$`)

// Run reports each Review that names no item and is in no item's review.
func (ReviewItem) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, l := range reviewsOf(c) {
		if _, ok := l.N.Get("itemReviewed"); !ok && !reviewNested.MatchString(l.Path) {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "this Review has no itemReviewed: name what it reviews, or put it in that item's review"))
		}
	}
	return out
}
