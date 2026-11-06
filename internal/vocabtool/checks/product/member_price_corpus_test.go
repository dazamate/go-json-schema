package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// A menu link to a rewards club (kasaustralia.com.au), a loyalty programme's
// terms (reliancedigital.in) and a membership's free delivery
// (officeworks.com.au) show no member price.
func TestMemberPriceNeedsAPrice(t *testing.T) {
	for _, body := range []string{
		`<p>REWARDS CLUB See More How it works BLOG SALE</p>`,
		`<p>Offers T&C RelianceOne Loyalty Program T&C Caution Notice</p>`,
		`<p>OnePass is a membership program. For $4 a month or $40 a year, members get free delivery on eligible orders.</p>`,
	} {
		audittest.Expect(t, product.MemberPrice{}, audittest.Input{Draft: noDraft, PageHTML: shopPageWith(body), PageURL: notesURL}, nil)
	}
	audittest.Expect(t, product.MemberPrice{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Club price $45.00, members only price $40</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page shows a member price ("Club price"): add a MemberProgram (name, hasTiers a MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPrice) as the business's hasMemberProgram, and give the Offer a UnitPriceSpecification of the member price with validForMemberTier that tier`},
	})
}
