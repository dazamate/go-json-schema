package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestAggregateOfferRequired(t *testing.T) {
	const need = ` (in "offers"): an AggregateOffer needs lowPrice and priceCurrency`
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"no range", `{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer","highPrice":2,"offerCount":3}}`, []audit.Issue{
			warn("$.offers", `Missing field "lowPrice"`+need),
			warn("$.offers", `Missing field "priceCurrency"`+need),
		}},
		{"no currency, in a list", `{"@type":"Product","name":"Box","offers":[{"@type":"AggregateOffer","lowPrice":1}]}`, []audit.Issue{
			warn("$.offers[0]", `Missing field "priceCurrency"`+need),
		}},
		{"an offer given by reference", `[{"@type":"Product","name":"Box","offers":{"@id":"#range"}},{"@type":"AggregateOffer","@id":"#range","highPrice":2}]`, []audit.Issue{
			warn("$.@graph[0].offers", `Missing field "lowPrice"`+need),
			warn("$.@graph[0].offers", `Missing field "priceCurrency"`+need),
		}},
		{"a ProductGroup's offers", `{"@type":"ProductGroup","name":"Box","offers":{"@type":"AggregateOffer"}}`, []audit.Issue{
			warn("$.offers", `Missing field "lowPrice"`+need),
			warn("$.offers", `Missing field "priceCurrency"`+need),
		}},
		{"a whole range", `{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer","lowPrice":1,"priceCurrency":"USD"}}`, nil},
		{"an Offer", `{"@type":"Product","name":"Box","offers":{"@type":"Offer"}}`, nil},
		{"an AggregateOffer outside a Product", `{"@type":"AggregateOffer","highPrice":2}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.AggregateOfferRequired{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
