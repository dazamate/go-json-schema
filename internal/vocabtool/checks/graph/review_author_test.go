package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestReviewAuthor(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"no author", `{"@context":"https://schema.org","@graph":[{"@type":"Review","reviewBody":"Great host","itemReviewed":{"@id":"https://x.test/#org"}}]}`, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: "this Review has no author: name who wrote it"},
		}},
		{"an author", `{"@type":"Review","reviewBody":"Great host","author":{"@type":"Person","name":"Bo Chan"}}`, nil},
		// a reference says nothing itself
		{"a bare reference", `{"@type":"Article","headline":"Tea","mentions":{"@type":"Review"}}`, nil},
		{"no Review", `{"@type":"Article","headline":"Tea"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, graph.ReviewAuthor{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
