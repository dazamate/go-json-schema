package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// kasaustralia.com.au's review widget is a div with role=button.
func TestReviewActionRoleButton(t *testing.T) {
	page := plainPageWith(`<div role="button" tabindex="0" aria-label="Be the first to write a review"><div tabindex="0">Be the first to write a review</div></div>`)
	audittest.Expect(t, product.ReviewAction{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
		{Path: "html:div[aria-label=\"Be the first to write a review\"]", Severity: audit.Warning, Message: `the page invites reviews ("Be the first to write a review"): add a ReviewAction (target the review form) to the Product's potentialAction`},
	})
}

// sublimecandles.co.uk and allrecipes.com share with an icon button that
// names no network.
func TestShareActionButton(t *testing.T) {
	audittest.Expect(t, product.ShareAction{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: plainPageWith(`<button type="button" aria-label="Share"><svg></svg></button>`)}, []audit.Issue{
		{Path: "html:button[aria-label=\"Share\"]", Severity: audit.Warning, Message: `the page has share buttons: add a ShareAction to the page's (or the Product's) potentialAction`},
	})
	audittest.Expect(t, product.ShareAction{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: plainPageWith(`<a href="/do-not-share">Do not share or sell my info</a><button>Shared &amp; Dedicated Hosting</button>`)}, nil)
}
