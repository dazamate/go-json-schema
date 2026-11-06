package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestOwnReview(t *testing.T) {
	const org = `{"@type":"Organization","@id":"https://x.test/#org","name":"Ann Lee Weddings"}`
	const site = `{"@type":"WebSite","publisher":{"@id":"https://x.test/#org"}}`
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"a review of the publisher", `{"@context":"https://schema.org","@graph":[` + org + `,` + site + `,
			{"@type":"Review","reviewBody":"Great host","itemReviewed":{"@id":"https://x.test/#org"}}]}`, []audit.Issue{
			{Path: "$.@graph[2]", Severity: audit.Info, Message: "a Review of the page's own business or person: the markup is valid, but Google shows no review stars for self-serving reviews"},
		}},
		{"a rating of the provider, by id", `{"@context":"https://schema.org","@graph":[
			{"@type":"Service","name":"Hosting","provider":"https://x.test/#org"},
			{"@type":"AggregateRating","ratingValue":4,"ratingCount":3,"itemReviewed":"https://x.test/#org"}]}`, []audit.Issue{
			{Path: "$.@graph[1]", Severity: audit.Info, Message: "a AggregateRating of the page's own business or person: the markup is valid, but Google shows no review stars for self-serving reviews"},
		}},
		{"a review of another business", `{"@context":"https://schema.org","@graph":[` + org + `,` + site + `,
			{"@type":"Review","reviewBody":"Great host","itemReviewed":{"@id":"https://y.test/#other"}}]}`, nil},
		{"no publisher or provider", `{"@context":"https://schema.org","@graph":[` + org + `,
			{"@type":"Review","reviewBody":"Great host","itemReviewed":{"@id":"https://x.test/#org"}}]}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, graph.OwnReview{}, audittest.Input{Draft: tc.draft}, tc.want) })
	}
}
