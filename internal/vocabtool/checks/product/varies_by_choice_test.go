package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestVariesByChoice(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", sizeSelect)
	colour := `{"@context":"https://schema.org","@type":"ProductGroup","productGroupID":"art-deco","variesBy":"https://schema.org/color"}`
	size := `{"@context":"https://schema.org","@type":"ProductGroup","productGroupID":"art-deco","variesBy":"https://schema.org/size"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a choice missing from variesBy", audittest.Input{Draft: colour, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `the page's Size choice is missing from variesBy: add https://schema.org/size`},
		}},
		{"variesBy names the choice", audittest.Input{Draft: size, PageHTML: page, PageURL: productURL}, nil},
		{"no ProductGroup", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, nil},
		{"no choice", audittest.Input{Draft: colour, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Size 6 7 8`, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.VariesByChoice{}, tc.in, tc.want) })
	}
}
