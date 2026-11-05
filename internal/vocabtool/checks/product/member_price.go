package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(MemberPrice{}) }

var memberPrice = regexp.MustCompile(`(?i)\bmember(?:s|ship)?['’]?\s+price\b|\bmembers?\s+(?:save|pay|get)\b|\b(?:rewards?|loyalty)\s+(?:club|program(?:me)?|members?)\b|\bclub members\b`)

// MemberPrice warns when a shop's page shows a member price and the draft has
// no MemberProgram: a price for members is an Offer price tied to the tier
// of the business's program.
type MemberPrice struct{}

// ID is the check's ID.
func (MemberPrice) ID() string { return "product/member-price" }

// Run reports the first words that show a member price or program.
func (MemberPrice) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() || c.Has("MemberProgram") {
		return nil
	}
	m := memberPrice.FindString(c.Page.Main)
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows a member price (%q): add a MemberProgram (name, hasTiers a MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPrice) as the business's hasMemberProgram, and give the Offer a UnitPriceSpecification of the member price with validForMemberTier that tier", m)}
}
