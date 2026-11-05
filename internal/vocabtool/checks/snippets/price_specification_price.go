package snippets

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(PriceSpecificationPrice{}) }

// PriceSpecificationPrice warns about a price specification under a Product's
// offer with no price: Google requires it there, and minPrice and maxPrice do
// not stand in for it. A range belongs on the offer alone, as an
// AggregateOffer's lowPrice and highPrice.
type PriceSpecificationPrice struct{}

// ID is the check's ID.
func (PriceSpecificationPrice) ID() string { return "snippets/price-specification-price" }

// Run reports each price specification with no price.
func (PriceSpecificationPrice) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, o := range offersOf(c) {
		for _, s := range o.specs {
			if c.Is(s.n, "PriceSpecification") && !has(s.n, "price") {
				out = append(out, audit.Issuef(s.path, audit.Warning, critical+`Missing field "price" (in "offers.priceSpecification"): Google requires price on a %s under a product's offers, and minPrice and maxPrice do not stand in for it. Give it price, or carry a range on the offer alone (an AggregateOffer's lowPrice and highPrice)`, jsonld.TypeNames(s.n)[0]))
			}
		}
	}
	return out
}
