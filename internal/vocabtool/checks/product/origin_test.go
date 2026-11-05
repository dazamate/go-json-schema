package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestOrigin(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", "<p>Handmade in Australia.</p>")
	carried := `{"@context":"https://schema.org","@type":"Product","countryOfOrigin":{"@type":"Country","name":"Australia"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"where it is made", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Handmade in Australia": set the Product's countryOfOrigin, a Country (name Australia)`},
		}},
		{"countryOfOrigin is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"a page that says nothing of where", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Origin{}, tc.in, tc.want) })
	}
}
