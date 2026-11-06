package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(WasPrice{}) }

var _ audit.PriceTaker = WasPrice{}

// WasPrice warns on a product page's was price that no StrikethroughPrice
// carries: Google shows a sale price beside the price it was. It takes every
// was price a shop's page shows, carried or not, for the sale checks, so the
// prices checks do not ask for them as items' prices.
type WasPrice struct{}

// ID is the check's ID.
func (WasPrice) ID() string { return "product/was-price" }

// Run reports the one was price a product page shows that the draft lacks.
func (WasPrice) Run(c *audit.Context) []audit.Issue {
	s, ok := saleOf(c)
	if !ok || len(s.missing) != 1 || c.Page.Kind != webpage.KindProduct {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows %s as a was price: add it to the Offer's priceSpecification as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice and price %s, beside the sale price (Google's sale-price markup)", s.missing[0], txt.DigitsOf(s.missing[0]))}
}

// TakesPrices takes every was price the page shows.
func (WasPrice) TakesPrices(c *audit.Context, _ audit.Taken) audit.Taken {
	s, ok := saleOf(c)
	if !ok {
		return nil
	}
	t := audit.Taken{}
	for _, w := range s.was {
		t.Take(w)
	}
	return t
}
