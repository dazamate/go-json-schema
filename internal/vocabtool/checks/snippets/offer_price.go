package snippets

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(OfferPrice{}) }

// OfferPrice warns about an Offer of a Product with no price, on itself or in
// one of its price specifications: Search Console marks the item invalid
// without one. An AggregateOffer is held to its range instead.
type OfferPrice struct{}

// ID is the check's ID.
func (OfferPrice) ID() string { return "snippets/offer-price" }

// Run reports each Offer with no price.
func (OfferPrice) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, o := range offersOf(c) {
		if plainOffer(c, o) && !has(o.n, "price") && !o.specHas("price") {
			out = append(out, audit.Issuef(o.path, audit.Warning, critical+`Missing field "price" (in "offers"): give the Offer price, or a priceSpecification with price`))
		}
	}
	return out
}

// plainOffer reports whether the offer is an Offer that is not an
// AggregateOffer: the kind held to a price, a currency and an availability.
func plainOffer(c *audit.Context, o offer) bool {
	return !c.Is(o.n, "AggregateOffer") && c.Is(o.n, "Offer")
}
