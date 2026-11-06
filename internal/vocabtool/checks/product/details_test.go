package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestDetails(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", "<ul><li>Chain Material: 925 Sterling Silver Filled</li><li>Weight: -</li></ul>")
	carried := `{"@context":"https://schema.org","@type":"Product","name":"Art Deco ring","material":"925 Sterling Silver Filled"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a detail the Product lacks", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page lists product details the markup lacks: Chain Material "925 Sterling Silver Filled" (material); set each on the Product`},
		}},
		{"the Product carries it", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Chain Material: 925 Sterling Silver Filled`, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Details{}, tc.in, tc.want) })
	}
}
