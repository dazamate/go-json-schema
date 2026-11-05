package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestOneOfAKind(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", "<p>This one-of-a-kind ring is carved by hand.</p>")
	carried := `{"@context":"https://schema.org","@type":"IndividualProduct","name":"Art Deco ring"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a one-of-a-kind piece", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "one-of-a-kind": type the Product IndividualProduct (with its serialNumber if it has one)`},
		}},
		{"an IndividualProduct is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"a piece made in numbers", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", "<p>Carved by hand.</p>"), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.OneOfAKind{}, tc.in, tc.want) })
	}
}
