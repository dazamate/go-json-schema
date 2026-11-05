package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestVariesByText(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", sizeSelect)
	text := `{"@context":"https://schema.org","@type":"ProductGroup","productGroupID":"art-deco","variesBy":"Size"}`
	url := `{"@context":"https://schema.org","@type":"ProductGroup","productGroupID":"art-deco","variesBy":"https://schema.org/size"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a variesBy in words", audittest.Input{Draft: text, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "$.variesBy", Severity: audit.Warning, Message: `variesBy "Size" is text: use https://schema.org/size`},
		}},
		{"a variesBy that is a property URL", audittest.Input{Draft: url, PageHTML: page, PageURL: productURL}, nil},
		{"no ProductGroup", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, nil},
		{"no choice", audittest.Input{Draft: text, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Size 6 7 8`, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.VariesByText{}, tc.in, tc.want) })
	}
}
