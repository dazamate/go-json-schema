package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// opalURL is a product page of the shop.
const opalURL = "https://harbour.test/products/opal-ring"

func TestCondition(t *testing.T) {
	page := shopPageWith("<p>A pre-owned ring in a damaged box.</p>")
	carried := `{"@context":"https://schema.org","@graph":[{"@type":"Offer","itemCondition":"https://schema.org/UsedCondition"},{"@type":"Offer","itemCondition":"https://schema.org/DamagedCondition"}]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a used ring in a damaged box", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: opalURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "pre-owned": set the Offer's itemCondition https://schema.org/UsedCondition (an OfferItemCondition)`},
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "damaged box": set the Offer's itemCondition https://schema.org/DamagedCondition (an OfferItemCondition)`},
		}},
		{"the conditions are there", audittest.Input{Draft: carried, PageHTML: page, PageURL: opalURL}, nil},
		{"no condition", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		// a pre-owned link in a menu of a contact page says nothing of a product
		{"a contact page", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Certified pre-owned phones. Unlocked phones.</p>"), PageURL: "https://harbour.test/support/contact-us"}, nil},
		{"a product page", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>A pre-owned ring.</p>"), PageURL: opalURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "pre-owned": set the Offer's itemCondition https://schema.org/UsedCondition (an OfferItemCondition)`},
		}},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Condition{}, tc.in, tc.want) })
	}
}
