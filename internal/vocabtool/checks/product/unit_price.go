package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(UnitPrice{}) }

var _ audit.PriceTaker = UnitPrice{}

// UnitPrice warns on a unit price a shop's page gives ("$4.50 per 100 g")
// when no UnitPriceSpecification has a referenceQuantity. It takes the price
// of each extra it and AddOn report, so the prices checks do not ask for
// them as items' prices.
type UnitPrice struct{}

// ID is the check's ID.
func (UnitPrice) ID() string { return "product/unit-price" }

// Run reports the first unit price the page gives.
func (UnitPrice) Run(c *audit.Context) []audit.Issue {
	unit, _ := extras(c)
	if unit == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page gives a unit price (%q): add a UnitPriceSpecification with that price and referenceQuantity (a QuantitativeValue with value and unitCode) to the Offer's priceSpecification", unit)}
}

// TakesPrices takes the unit price and the add-on's price the extras checks
// report.
func (UnitPrice) TakesPrices(c *audit.Context, _ audit.Taken) audit.Taken {
	unit, extra := extras(c)
	t := audit.Taken{}
	t.Take(unit)
	t.Take(extra)
	return t
}
