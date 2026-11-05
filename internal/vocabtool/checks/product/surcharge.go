package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Surcharge{}) }

var surcharge = regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*%\s*(?:card\s+|credit card\s+)?surcharge\b|\b(?:incur|attract)s?\s+an?\s+\d+(?:\.\d+)?\s*%\s*surcharge\b|\bsurcharge of \d+(?:\.\d+)?\s*%`)

// Surcharge warns on a card surcharge a shop's page states that no
// PaymentChargeSpecification carries.
type Surcharge struct{}

// ID is the check's ID.
func (Surcharge) ID() string { return "product/surcharge" }

// Run reports the first surcharge the page states.
func (Surcharge) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	m := surcharge.FindString(c.Page.Main)
	if m == "" || c.Has("PaymentChargeSpecification") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: add a PaymentChargeSpecification (appliesToPaymentMethod the cards it applies to, and the charge) to the Offer's priceSpecification", m)}
}
