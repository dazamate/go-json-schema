package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestProductGroup(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", sizeSelect)
	carried := `{"@context":"https://schema.org","@type":"ProductGroup","productGroupID":"art-deco","variesBy":"https://schema.org/size"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a choice of size", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page offers a choice of Size (6, 7, 8): make the product a ProductGroup (productGroupID, variesBy https://schema.org/size) with a Product for each choice in hasVariant, each with its size and its own Offer`},
		}},
		{"a ProductGroup is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"no choice", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Size 6 7 8`, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.ProductGroup{}, tc.in, tc.want) })
	}
}
