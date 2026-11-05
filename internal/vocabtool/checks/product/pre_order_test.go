package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestPreOrder(t *testing.T) {
	page := shopPageWith("<p>Pre-order now for May.</p>")
	both := `{"@context":"https://schema.org","@graph":[{"@type":"Offer","availability":"https://schema.org/PreOrder"},{"@type":"PreOrderAction"}]}`
	offerOnly := `{"@context":"https://schema.org","@type":"Offer","availability":"https://schema.org/PreOrder"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a pre-order", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page takes pre-orders ("Pre-order now"): set the Offer's availability https://schema.org/PreOrder and add a PreOrderAction as its potentialAction (target the order form), with availabilityStarts when the page gives a date`},
		}},
		{"a PreOrder availability and no action", audittest.Input{Draft: offerOnly, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page takes pre-orders ("Pre-order now"): set the Offer's availability https://schema.org/PreOrder and add a PreOrderAction as its potentialAction (target the order form), with availabilityStarts when the page gives a date`},
		}},
		{"the availability and the action are there", audittest.Input{Draft: both, PageHTML: page, PageURL: notesURL}, nil},
		{"no pre-order", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.PreOrder{}, tc.in, tc.want) })
	}
}
