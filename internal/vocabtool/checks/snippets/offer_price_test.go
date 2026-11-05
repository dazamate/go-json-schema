package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestOfferPrice(t *testing.T) {
	const msg = `Missing field "price" (in "offers"): give the Offer price, or a priceSpecification with price`
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"no price anywhere", `{"@type":"Product","name":"Box","offers":[{"@type":"Offer","priceCurrency":"USD","priceSpecification":{"@type":"UnitPriceSpecification","minPrice":1,"maxPrice":2}}]}`, []audit.Issue{
			warn("$.offers[0]", msg),
		}},
		{"a single offer", `{"@type":"Product","name":"Box","offers":{"@type":"Offer"}}`, []audit.Issue{warn("$.offers", msg)}},
		{"an offer given by reference", `[{"@type":"Product","name":"Box","offers":{"@id":"#o"}},{"@type":"Offer","@id":"#o","priceCurrency":"USD"}]`, []audit.Issue{
			warn("$.@graph[0].offers", msg),
		}},
		{"a ProductGroup's offer", `{"@type":"ProductGroup","name":"Box","offers":{"@type":"Offer"}}`, []audit.Issue{warn("$.offers", msg)}},
		{"a price", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1}}`, nil},
		{"priced through its priceSpecification", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","priceSpecification":{"@type":"UnitPriceSpecification","price":1.5}}}`, nil},
		{"priced through a priceSpecification given by reference", `[{"@type":"Product","name":"Box","offers":{"@type":"Offer","priceSpecification":{"@id":"#s"}}},{"@type":"UnitPriceSpecification","@id":"#s","price":1.5}]`, nil},
		// the range check asks for its lowPrice
		{"an AggregateOffer", `{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer"}}`, nil},
		{"an Offer outside a Product", `{"@type":"Offer"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.OfferPrice{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
