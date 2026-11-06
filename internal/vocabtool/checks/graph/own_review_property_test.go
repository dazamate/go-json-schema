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
}
