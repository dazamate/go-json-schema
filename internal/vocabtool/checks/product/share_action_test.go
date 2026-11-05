package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestShareAction(t *testing.T) {
	page := plainPageWith("<p><a href=\"/reviews/new\">Write a review</a> <a href=\"https://www.facebook.com/sharer/sharer.php?u=x\">Share</a> <a href=\"/wishlist\">Add to wishlist</a> <a href=\"/track\">Track your order</a></p>")
	carried := `{"@context":"https://schema.org","@type":"ShareAction"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a link that invites it", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body > main > p > a:nth-of-type(2)", Severity: audit.Warning, Message: `the page has share buttons: add a ShareAction to the page's (or the Product's) potentialAction`},
		}},
		{"the action is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"a page with no such link", audittest.Input{Draft: noDraft, PageHTML: plainPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Share`, PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.ShareAction{}, tc.in, tc.want) })
	}
}
