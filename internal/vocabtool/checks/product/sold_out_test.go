package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestSoldOut(t *testing.T) {
	page := shopPageWith("<p>Hide sold out</p><p>Sold out</p>")
	filter := shopPageWith("<p>Hide sold out</p>")
	carried := `{"@context":"https://schema.org","@type":"Offer","availability":"https://schema.org/OutOfStock"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a sold-out product", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page shows "Sold out": set that product's Offer availability https://schema.org/SoldOut (or OutOfStock)`},
		}},
		{"OutOfStock is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"a filter that hides sold out", audittest.Input{Draft: noDraft, PageHTML: filter, PageURL: notesURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.SoldOut{}, tc.in, tc.want) })
	}
}
