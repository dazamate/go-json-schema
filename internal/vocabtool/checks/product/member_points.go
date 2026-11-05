package product

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(MemberPoints{}) }

var pointsEarned = regexp.MustCompile(`(?i)\bearn\s+(\d[\d,]*)\s+(?:reward\s+|loyalty\s+|bonus\s+)?points\b`)

// MemberPoints warns when a shop's page says a purchase earns loyalty points
// and no node has membershipPointsEarned.
type MemberPoints struct{}

// ID is the check's ID.
func (MemberPoints) ID() string { return "product/member-points" }

// Run reports the first points offer the page states.
func (MemberPoints) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() || jsonld.HasKey(c.Nodes, "membershipPointsEarned") {
		return nil
	}
	m := pointsEarned.FindStringSubmatch(c.Page.Main)
	if m == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set membershipPointsEarned %s on the Offer's UnitPriceSpecification (validForMemberTier the MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPoints)", m[0], strings.ReplaceAll(m[1], ",", ""))}
}
