package snippets

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(OfferPriceCurrency{}) }

// OfferPriceCurrency notes an Offer of a Product with no priceCurrency, on
// itself or in one of its price specifications: Search Console only warns on
// it. An AggregateOffer is held to its range instead.
type OfferPriceCurrency struct{}

// ID is the check's ID.
func (OfferPriceCurrency) ID() string { return "snippets/offer-price-currency" }

// Run reports each Offer with no currency.
func (OfferPriceCurrency) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, o := range offersOf(c) {
		if plainOffer(c, o) && !has(o.n, "priceCurrency") && !o.specHas("priceCurrency") {
			out = append(out, audit.Issuef(o.path, audit.Info, nonCritical+`Missing field "priceCurrency" (in "offers"): give the Offer's currency as a three-letter ISO 4217 code`))
		}
	}
	return out
}
