package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Discount{}) }

var discount = regexp.MustCompile(`(?i)\b\d{1,2}% off\b|\bsave \d{1,2}%`)

// Discount warns on a shop's page that gives a discount ("20% off") and shows
// no was price: the markup can only carry a sale as the price it was.
type Discount struct{}

// ID is the check's ID.
func (Discount) ID() string { return "product/discount" }

// Run reports the first discount when neither the page nor the draft has a
// was price.
func (Discount) Run(c *audit.Context) []audit.Issue {
	s, ok := saleOf(c)
	if !ok || len(s.was) > 0 || len(s.struck) > 0 {
		return nil
	}
	m := discount.FindString(c.Page.Main)
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows a discount (%q) but no was price the markup can carry: show the was price, and mark it up as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice", m)}
}
