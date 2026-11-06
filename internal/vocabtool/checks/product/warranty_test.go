package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestWarranty(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", "<p>Comes with a 2 year warranty.</p>")
	carried := `{"@context":"https://schema.org","@type":"WarrantyPromise","durationOfWarranty":{"@type":"QuantitativeValue","value":2,"unitCode":"ANN"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a warranty", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page promises "2 year warranty": set the Offer's warranty, a WarrantyPromise (durationOfWarranty a QuantitativeValue with unitCode ANN, warrantyScope)`},
		}},
		{"a WarrantyPromise is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"no warranty", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Warranty{}, tc.in, tc.want) })
	}
}
