package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestProductSet(t *testing.T) {
	page := productPageWith("Art Deco ring set, 3 pieces", "Harbour Jewellers", "")
	carried := `{"@context":"https://schema.org","@type":"ProductCollection","name":"Art Deco ring set"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a set of pieces", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page sells "Art Deco ring set, 3 pieces", a set of 3: type its Product a ProductCollection and list the pieces in includesObject, a TypeAndQuantityNode for each (typeOfGood, amountOfThisGood)`},
		}},
		{"a ProductCollection is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"a single piece", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.ProductSet{}, tc.in, tc.want) })
	}
}
