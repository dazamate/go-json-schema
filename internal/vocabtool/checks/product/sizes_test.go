package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestSizes(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", "<p>Sizes: 6 - 16mm, 7 - 17mm inner diameter.</p>")
	carried := `{"@context":"https://schema.org","@type":"SizeSpecification","name":"6"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"sizes with measurements", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives sizes with their measurements (6: 16mm, 7: 17mm): make each variant's size a SizeSpecification (name, hasMeasurement a QuantitativeValue with value and unitCode, sizeSystem a WearableSizeSystemEnumeration member such as WearableSizeSystemUS, sizeGroup a WearableSizeGroupEnumeration member)`},
		}},
		{"a SizeSpecification is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"one measurement only", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", "<p>Size 6 - 16mm inner diameter.</p>"), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Sizes{}, tc.in, tc.want) })
	}
}
