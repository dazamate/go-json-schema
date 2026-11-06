package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ShareAction{}) }

// ShareAction warns when a page has share buttons and the draft has no
// ShareAction: the buttons are an action on the page or its Product.
type ShareAction struct{}

// ID is the check's ID.
func (ShareAction) ID() string { return "product/share-action" }

// Run reports the first share button, at its element.
func (ShareAction) Run(c *audit.Context) []audit.Issue {
	l := engagementLink(c, kindShare)
	if l == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(l.path, audit.Warning, "the page has share buttons: add a ShareAction to the page's (or the Product's) potentialAction")}
}
