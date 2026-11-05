package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestOwnReviewProperty(t *testing.T) {
	const site = `{"@type":"WebSite","publisher":{"@id":"https://x.test/#org"}}`
	const stars = "the markup is valid, but Google shows no review stars for self-serving reviews"
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"the publisher carries a review and a rating", `{"@context":"https://schema.org","@graph":[` + site + `,
			{"@type":"Organization","@id":"https://x.test/#org","name":"Ann Lee Weddings",
			 "aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3},
			 "review":{"@type":"Review","author":{"@type":"Person","name":"Bo Chan"}}}]}`, []audit.Issue{
			{Path: "$.@graph[1].review", Severity: audit.Info, Message: "the page's own business or person carries review: " + stars},
			{Path: "$.@graph[1].aggregateRating", Severity: audit.Info, Message: "the page's own business or person carries aggregateRating: " + stars},
		}},
		{"another business carries one", `{"@context":"https://schema.org","@graph":[` + site + `,
			{"@type":"LocalBusiness","@id":"https://y.test/#other","name":"Other","aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}}]}`, nil},
		{"no publisher or provider", `{"@type":"Organization","@id":"https://x.test/#org","name":"Ann Lee Weddings","aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, graph.OwnReviewProperty{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
	for _, tc := range []struct {
		name, draft, url string
		want             []audit.Issue
	}{
		// mooreheaddentistry.com: the site's own practice, at its home page,
		// rates each of its locations
		{"the site's business and its departments", `{"@context":"https://schema.org","@type":"Organization","@id":"https://www.mooreheaddentistry.com/#organization","name":"Moorehead Dentistry","url":"https://www.mooreheaddentistry.com/",
			"department":[{"@type":"Dentist","@id":"https://www.mooreheaddentistry.com/contact/blue-ash-dentistry/#location","name":"Moorehead Dentistry - Blue Ash","url":"https://www.mooreheaddentistry.com/contact/blue-ash-dentistry/","aggregateRating":{"@type":"AggregateRating","ratingValue":"4.9","reviewCount":"1079"}},
			{"@type":"Dentist","@id":"https://www.mooreheaddentistry.com/contact/batavia-dentistry/#location","name":"Moorehead Dentistry - Batavia","url":"https://www.mooreheaddentistry.com/contact/batavia-dentistry/","aggregateRating":{"@type":"AggregateRating","ratingValue":"4.9","reviewCount":"1112"}}]}`,
			"https://www.mooreheaddentistry.com/financial/office-fees/", []audit.Issue{
				{Path: "$.department[0].aggregateRating", Severity: audit.Info, Message: "the page's own business or person carries aggregateRating: " + stars},
				{Path: "$.department[1].aggregateRating", Severity: audit.Info, Message: "the page's own business or person carries aggregateRating: " + stars},
			}},
		// loveitlandscaping.com.au: a business named as the site rates itself
		{"a business named as the site", `{"@context":"https://schema.org","@graph":[
			{"@type":"WebSite","@id":"https://loveitlandscaping.com.au/#website","url":"https://loveitlandscaping.com.au","name":"Love It Landscaping"},
			{"@type":"Organization","name":"Love It Landscaping","aggregateRating":{"@type":"AggregateRating","ratingValue":4.6,"reviewCount":36}}]}`,
			"https://loveitlandscaping.com.au/landscaping-costs-in-melbourne-a-realistic-2026-guide/", []audit.Issue{
				{Path: "$.@graph[1].aggregateRating", Severity: audit.Info, Message: "the page's own business or person carries aggregateRating: " + stars},
			}},
		// a directory rates the businesses it lists on its own pages
		{"a listed business on the site", `{"@context":"https://schema.org","@type":"LocalBusiness","name":"Acme Plumbing","url":"https://x.test/plumbers/acme/","aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}}`,
			"https://x.test/plumbers/acme/", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, graph.OwnReviewProperty{}, audittest.Input{Draft: tc.draft, PageURL: tc.url, PageText: "Reviews."}, tc.want)
		})
	}
}
