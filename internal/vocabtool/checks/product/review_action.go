package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ReviewAction{}) }

var writeReview = regexp.MustCompile(`(?i)^(?:write|leave|add) a review$`)

// ReviewAction warns when a page invites reviews ("Write a review") and the
// draft has no ReviewAction: the invitation is an action on the Product.
type ReviewAction struct{}

// ID is the check's ID.
func (ReviewAction) ID() string { return "product/review-action" }

// Run reports the first write-a-review link or button, at its element.
func (ReviewAction) Run(c *audit.Context) []audit.Issue {
	if c.Has("ReviewAction") {
		return nil
	}
	l := firstLink(c, func(l link) bool { return writeReview.MatchString(l.text) })
	if l == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(l.path, audit.Warning, "the page invites reviews (%q): add a ReviewAction (target the review form) to the Product's potentialAction", l.text)}
}
