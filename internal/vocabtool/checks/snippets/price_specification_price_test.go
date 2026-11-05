package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestPriceSpecificationPrice(t *testing.T) {
	msg := func(ty string) string {
		return `Missing field "price" (in "offers.priceSpecification"): Google requires price on a ` + ty + ` under a product's offers, and minPrice and maxPrice do not stand in for it. Give it price, or carry a range on the offer alone (an AggregateOffer's lowPrice and highPrice)`
	}
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"a range in place of a price", `{"@type":"Product","name":"Box","offers":[{"@type":"Offer","priceSpecification":{"@type":"UnitPriceSpecification","minPrice":1,"maxPrice":2}}]}`, []audit.Issue{
			warn("$.offers[0].priceSpecification", msg("UnitPriceSpecification")),
		}},
		{"a list of specifications", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1,"priceSpecification":[{"@type":"PriceSpecification","price":1},{"@type":"PriceSpecification"}]}}`, []audit.Issue{
			warn("$.offers.priceSpecification[1]", msg("PriceSpecification")),
		}},
		// the path counts the raw list, a non-node entry included
		{"a list with a non-node first", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1,"priceSpecification":["x",{"@type":"PriceSpecification"}]}}`, []audit.Issue{
			warn("$.offers.priceSpecification[1]", msg("PriceSpecification")),
		}},
		{"a specification given by reference", `[{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer","priceSpecification":{"@id":"#s"}}},{"@type":"UnitPriceSpecification","@id":"#s","minPrice":1}]`, []audit.Issue{
			warn("$.@graph[0].offers.priceSpecification", msg("UnitPriceSpecification")),
		}},
		{"a price", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","priceSpecification":{"@type":"UnitPriceSpecification","price":1.5}}}`, nil},
		{"a node that is no PriceSpecification", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","priceSpecification":{"@type":"QuantitativeValue"}}}`, nil},
		{"a specification outside a Product's offers", `{"@type":"Offer","priceSpecification":{"@type":"UnitPriceSpecification","minPrice":1}}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.PriceSpecificationPrice{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
