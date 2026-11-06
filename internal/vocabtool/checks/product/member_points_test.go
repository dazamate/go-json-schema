package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestMemberPoints(t *testing.T) {
	page := shopPageWith("<p>Earn 1,200 points on this order.</p>")
	carried := `{"@context":"https://schema.org","@type":"UnitPriceSpecification","membershipPointsEarned":1200}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"points earned", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Earn 1,200 points": set membershipPointsEarned 1200 on the Offer's UnitPriceSpecification (validForMemberTier the MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPoints)`},
		}},
		{"membershipPointsEarned is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"no points", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.MemberPoints{}, tc.in, tc.want) })
	}
}
