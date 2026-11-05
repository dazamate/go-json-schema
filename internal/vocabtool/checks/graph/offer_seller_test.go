package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestOfferSeller(t *testing.T) {
	const draft = `{"@context":"https://schema.org","@type":"Organization","name":"Ann Lee Weddings",
		"makesOffer":{"@type":"Offer","itemOffered":{"@type":"Service","name":"Hosting"}}}`
	fires := []audit.Issue{{Path: "$.makesOffer", Severity: audit.Warning, Message: "this Offer names no seller: set seller (or offeredBy) to the business that makes it"}}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"an Offer with no seller", draft, audittest.Input{PageHTML: teaPage}, fires},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, fires},
		// the markup's own faults are held to a page search engines skip too
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"a seller", `{"@type":"Offer","seller":{"@id":"https://x.test/#org"}}`, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"offeredBy", `{"@type":"Offer","offeredBy":{"@id":"https://x.test/#org"}}`, audittest.Input{PageText: "Tea ceremony."}, nil},
		// it gathers many sellers' offers
		{"an AggregateOffer", `{"@type":"AggregateOffer","lowPrice":1}`, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, graph.OfferSeller{}, tc.in, tc.want)
		})
	}
}
