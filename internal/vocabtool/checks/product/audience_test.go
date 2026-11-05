package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestAudience(t *testing.T) {
	page := productPageWith("Women's opal ring", "Harbour Jewellers", "")
	forHim := productPageWith("Signet ring", "Signet ring for him", "")
	carried := `{"@context":"https://schema.org","@type":"PeopleAudience","suggestedGender":"https://schema.org/Female"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a product named for women", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "Women's ": set its audience, a PeopleAudience (suggestedGender https://schema.org/Female, a GenderType)`},
		}},
		{"a title that ends with the audience", audittest.Input{Draft: noDraft, PageHTML: forHim, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "for him": set its audience, a PeopleAudience (suggestedGender https://schema.org/Male, a GenderType)`},
		}},
		{"a PeopleAudience is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"a product named for nobody", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Opal ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Audience{}, tc.in, tc.want) })
	}
}
