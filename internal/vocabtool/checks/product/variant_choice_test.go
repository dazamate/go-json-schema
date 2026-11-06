package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestVariantChoice(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", sizeSelect)
	bare := `{"@context":"https://schema.org","@type":"ProductGroup","productGroupID":"art-deco","variesBy":"https://schema.org/size","hasVariant":[{"@type":"Product","name":"Art Deco ring - 6"},{"@type":"Product","name":"Art Deco ring - 7","size":"7"}]}`
	sized := `{"@context":"https://schema.org","@type":"ProductGroup","productGroupID":"art-deco","variesBy":"https://schema.org/size","hasVariant":[{"@type":"Product","name":"Art Deco ring - 6","size":"6"}]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a variant with no size", audittest.Input{Draft: bare, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "$.hasVariant[0]", Severity: audit.Warning, Message: `this variant has no size: set size "6"`},
		}},
		{"every variant has its size", audittest.Input{Draft: sized, PageHTML: page, PageURL: productURL}, nil},
		{"no ProductGroup", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, nil},
		{"no choice", audittest.Input{Draft: bare, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Size 6 7 8`, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.VariantChoice{}, tc.in, tc.want) })
	}
}
