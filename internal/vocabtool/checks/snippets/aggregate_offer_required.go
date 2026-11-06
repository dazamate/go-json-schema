package snippets

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(AggregateOfferRequired{}) }

// AggregateOfferRequired warns about an AggregateOffer of a Product that lacks
// lowPrice or priceCurrency: Search Console marks the item invalid without
// them, as a price range has no bottom or currency to show.
type AggregateOfferRequired struct{}

// ID is the check's ID.
func (AggregateOfferRequired) ID() string { return "snippets/aggregate-offer-required" }

// Run reports each field a Product's AggregateOffer lacks.
func (AggregateOfferRequired) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, o := range offersOf(c) {
		if !c.Is(o.n, "AggregateOffer") {
			continue
		}
		for _, k := range []string{"lowPrice", "priceCurrency"} {
			if !has(o.n, k) {
				out = append(out, audit.Issuef(o.path, audit.Warning, critical+`Missing field %q (in "offers"): an AggregateOffer needs lowPrice and priceCurrency`, k))
			}
		}
	}
	return out
}
