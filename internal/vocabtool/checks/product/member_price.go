package product

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(MemberPrice{}) }

var (
	memberPrice = regexp.MustCompile(`(?i)\b(?:member(?:s|ship)?|rewards?|loyalty|club|vip|insider)['’]?\s+(?:only\s+)?pric(?:e|es|ing)\b|\bmembers?\s+(?:save|pay|get)\b[^.]{0,40}`)
	// memberOffer is what a "members get ..." sentence must hold to be about a
	// price: a percentage or an amount, no free delivery or early access.
	memberOffer = regexp.MustCompile(`\d\s*%|` + txt.PriceText)
)

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
	m := ""
	for _, f := range memberPrice.FindAllString(c.Page.Main, -1) {
		if strings.Contains(strings.ToLower(f), "pric") {
			m = f
			break
		}
		if memberOffer.MatchString(f) {
			m = f[:strings.IndexAny(f, "0123456789$£€₹")]
			m = strings.TrimSpace(m)
			break
		}
	}
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows a member price (%q): add a MemberProgram (name, hasTiers a MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPrice) as the business's hasMemberProgram, and give the Offer a UnitPriceSpecification of the member price with validForMemberTier that tier", m)}
}
