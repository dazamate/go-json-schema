package snippets

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(OfferAvailability{}) }

// OfferAvailability notes an Offer of a Product with no availability: Search
// Console only warns on it, and stock is a fact the page may not give. An
// AggregateOffer is left out.
type OfferAvailability struct{}

// ID is the check's ID.
func (OfferAvailability) ID() string { return "snippets/offer-availability" }

// Run reports each Offer with no availability.
func (OfferAvailability) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, o := range offersOf(c) {
		if plainOffer(c, o) && !has(o.n, "availability") {
			out = append(out, audit.Issuef(o.path, audit.Info, nonCritical+`Missing field "availability" (in "offers"): give the Offer's availability, an ItemAvailability member such as InStock`))
		}
	}
	return out
}
