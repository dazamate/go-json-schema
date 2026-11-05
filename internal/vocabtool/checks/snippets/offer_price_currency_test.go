package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestOfferPriceCurrency(t *testing.T) {
	const msg = `Missing field "priceCurrency" (in "offers"): give the Offer's currency as a three-letter ISO 4217 code`
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"no currency", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1}}`, []audit.Issue{info("$.offers", msg)}},
		{"in a list", `{"@type":"Product","name":"Box","offers":[{"@type":"Offer","price":1,"priceCurrency":"USD"},{"@type":"Offer","price":2}]}`, []audit.Issue{info("$.offers[1]", msg)}},
		{"a currency", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1,"priceCurrency":"USD"}}`, nil},
		{"a currency in the priceSpecification", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","priceSpecification":{"@type":"UnitPriceSpecification","price":1.5,"priceCurrency":"USD"}}}`, nil},
		// the range check asks for its priceCurrency
		{"an AggregateOffer", `{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer"}}`, nil},
		{"an Offer outside a Product", `{"@type":"Offer"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.OfferPriceCurrency{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
