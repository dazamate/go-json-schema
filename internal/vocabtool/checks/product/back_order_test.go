package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestBackOrder(t *testing.T) {
	page := shopPageWith("<p>Available on back-order.</p>")
	carried := `{"@context":"https://schema.org","@type":"Offer","availability":"https://schema.org/BackOrder"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a back order", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "back-order": set the Offer's availability https://schema.org/BackOrder`},
		}},
		{"BackOrder is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"no back order", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.BackOrder{}, tc.in, tc.want) })
	}
}
