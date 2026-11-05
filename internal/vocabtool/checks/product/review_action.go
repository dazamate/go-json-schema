package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ReviewAction{}) }

var writeReview = regexp.MustCompile(`(?i)^(?:be the first to )?(?:write|leave|add|submit) (?:a|your) (?:customer |product )?review$`)

// ReviewAction warns when a page invites reviews ("Write a review") and the
// draft has no ReviewAction: the invitation is an action on the Product.
type ReviewAction struct{}

// ID is the check's ID.
func (ReviewAction) ID() string { return "product/review-action" }

// Run reports the first write-a-review link or button, at its element.
func (ReviewAction) Run(c *audit.Context) []audit.Issue {
	l := engagementLink(c, kindReview)
	if l == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(l.path, audit.Warning, "the page invites reviews (%q): add a ReviewAction (target the review form) to the Product's potentialAction", l.text)}
}
