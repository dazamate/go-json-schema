package costs

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(OfferPrices{}) }

// OfferPrices warns on the prices a shop's page shows that no price property
// carries and no shop check reports (was prices, shipping rates, product
// cards): each is the price of an item it sells, an Offer's price.
type OfferPrices struct{}

// ID is the check's ID.
func (OfferPrices) ID() string { return "costs/offer-prices" }

// Run reports the prices left over in one issue.
func (o OfferPrices) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	missing := unmarked(c, o.ID())
	if len(missing) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows prices (%s) that no price property carries: mark each up on the item it is the price of, as an Offer's price or, for a range, an AggregateOffer's lowPrice and highPrice, with priceCurrency and offerCount", strings.Join(missing, ", "))}
}
