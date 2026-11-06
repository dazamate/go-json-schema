package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestReviewItem(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"no itemReviewed", `{"@context":"https://schema.org","@graph":[{"@type":"Review","reviewBody":"Great host","author":"Bo Chan"}]}`, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: "this Review has no itemReviewed: name what it reviews, or put it in that item's review"},
		}},
		{"an itemReviewed", `{"@type":"Review","reviewBody":"Great host","itemReviewed":{"@id":"https://x.test/#org"}}`, nil},
		{"nested in the item's review", `{"@type":"Product","name":"Box","review":{"@type":"Review","author":"Bo Chan","reviewBody":"Sturdy."}}`, nil},
		{"nested in a list of the item's reviews", `{"@type":"Product","name":"Box","review":[{"@type":"Review","author":"Bo Chan","reviewBody":"Sturdy."}]}`, nil},
		{"a bare reference", `{"@type":"Article","headline":"Tea","mentions":{"@type":"Review"}}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, graph.ReviewItem{}, audittest.Input{Draft: tc.draft}, tc.want) })
	}
}
