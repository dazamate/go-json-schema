package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestAggregateOfferRecommended(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		// what Search Console flagged on liftmyboxes.com's box estimator
		{"a price range without offerCount", `{"@type":"Product","name":"Small moving box","offers":{"@type":"AggregateOffer","lowPrice":1.12,"highPrice":2.08,"priceCurrency":"USD"}}`, []audit.Issue{
			info("$.offers", `Missing field "offerCount" (in "offers"): give the AggregateOffer offerCount, how many offers the range spans: the sellers or sources it comes from`),
		}},
		{"a range without its top", `{"@type":"Product","name":"Box","offers":[{"@type":"AggregateOffer","lowPrice":1,"offerCount":3}]}`, []audit.Issue{
			info("$.offers[0]", `Missing field "highPrice" (in "offers"): give the AggregateOffer highPrice, the highest price the range reaches`),
		}},
		{"neither", `{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer"}}`, []audit.Issue{
			info("$.offers", `Missing field "highPrice" (in "offers"): give the AggregateOffer highPrice, the highest price the range reaches`),
			info("$.offers", `Missing field "offerCount" (in "offers"): give the AggregateOffer offerCount, how many offers the range spans: the sellers or sources it comes from`),
		}},
		{"offers given by reference", `[{"@type":"Product","name":"Box","offers":{"@id":"#range"}},{"@type":"AggregateOffer","@id":"#range","lowPrice":1,"highPrice":2,"priceCurrency":"USD"}]`, []audit.Issue{
			info("$.@graph[0].offers", `Missing field "offerCount" (in "offers"): give the AggregateOffer offerCount, how many offers the range spans: the sellers or sources it comes from`),
		}},
		{"a whole range", `{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer","highPrice":2,"offerCount":3}}`, nil},
		{"an Offer", `{"@type":"Product","name":"Box","offers":{"@type":"Offer"}}`, nil},
		{"an AggregateOffer outside a Product", `{"@type":"AggregateOffer"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.AggregateOfferRecommended{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
