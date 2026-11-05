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
		// the match keeps the space after the word, which the possessive trim
		// must see past
		{"a product named for men", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Men's leather wallet", "Harbour Jewellers", ""), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "Men's ": set its audience, a PeopleAudience (suggestedGender https://schema.org/Male, a GenderType)`},
		}},
		{"a product named for men, with a curly apostrophe", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Men’s leather wallet", "Harbour Jewellers", ""), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "Men’s ": set its audience, a PeopleAudience (suggestedGender https://schema.org/Male, a GenderType)`},
		}},
		{"a unisex product", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Unisex leather wallet", "Harbour Jewellers", ""), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "Unisex ": set its audience, a PeopleAudience (suggestedGender unisex)`},
		}},
		{"a product named for boys", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Boys' leather wallet", "Harbour Jewellers", ""), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "Boys' ": set its audience, a PeopleAudience (suggestedGender https://schema.org/Male, a GenderType)`},
		}},
		{"a product named for children", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Children's leather wallet", "Harbour Jewellers", ""), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "Children's ": set its audience, a PeopleAudience (suggestedMinAge and suggestedMaxAge)`},
		}},
		{"a product named for babies", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Babies' leather wallet", "Harbour Jewellers", ""), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the product is named for "Babies'": set its audience, a PeopleAudience (suggestedMinAge and suggestedMaxAge)`},
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
