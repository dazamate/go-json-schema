package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestOfferAvailability(t *testing.T) {
	const msg = `Missing field "availability" (in "offers"): give the Offer's availability, an ItemAvailability member such as InStock`
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"no availability", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1}}`, []audit.Issue{info("$.offers", msg)}},
		{"availability", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1,"availability":"InStock"}}`, nil},
		{"an AggregateOffer", `{"@type":"Product","name":"Box","offers":{"@type":"AggregateOffer"}}`, nil},
		{"an Offer outside a Product", `{"@type":"Offer"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.OfferAvailability{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
