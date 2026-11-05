package costs

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Prices{}) }

// Prices warns on the prices a site that sells nothing shows that no price
// property carries and no other check reports: they are what others charge,
// so each is a MonetaryAmount, and an Offer only for what the business
// itself sells.
type Prices struct{}

// ID is the check's ID.
func (Prices) ID() string { return "costs/prices" }

// Run reports the prices left over in one issue.
func (p Prices) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Sells() {
		return nil
	}
	missing := unmarked(c, p.ID())
	if len(missing) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows prices (%s) that no price property carries: mark each up on the thing it is the price of, as a MonetaryAmount (value, or minValue and maxValue, currency) for a cost or estimate of what others charge, or an Offer's price only for what the business itself sells", strings.Join(missing, ", "))}
}
