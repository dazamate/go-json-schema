package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestMemberPrice(t *testing.T) {
	page := shopPageWith("<p>Members save 10% on every ring.</p>")
	carried := `{"@context":"https://schema.org","@type":"MemberProgram","name":"Harbour Club"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a member price", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page shows a member price ("Members save"): add a MemberProgram (name, hasTiers a MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPrice) as the business's hasMemberProgram, and give the Offer a UnitPriceSpecification of the member price with validForMemberTier that tier`},
		}},
		{"a MemberProgram is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"no member price", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.MemberPrice{}, tc.in, tc.want) })
	}
}
