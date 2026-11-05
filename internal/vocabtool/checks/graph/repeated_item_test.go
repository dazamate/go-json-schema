package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestRepeatedItem(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"an inline node repeats a node with an @id", `{"@context":"https://schema.org","@graph":[
			{"@type":"City","@id":"https://x.test/#sydney","name":"Sydney","sameAs":["https://www.wikidata.org/wiki/Q3130"]},
			{"@type":"WebPage","@id":"https://x.test/#webpage",
			 "about":[{"@type":"City","name":"Sydney","sameAs":["https://en.wikipedia.org/wiki/Sydney","https://www.wikidata.org/wiki/Q3130"]},
			          {"@id":"https://x.test/#sydney"},
			          {"@type":"Thing","name":"Wedding","sameAs":"https://www.wikidata.org/wiki/Q49836"}]}]}`, []audit.Issue{
			{Path: "$.@graph[1].about[0]", Severity: audit.Warning, Message: `same Wikidata item (Q3130) as the node https://x.test/#sydney: put {"@id": "https://x.test/#sydney"} here instead of repeating it`},
		}},
		{"a node with an @id says its item once", `{"@context":"https://schema.org","@graph":[
			{"@type":"City","@id":"https://x.test/#sydney","name":"Sydney","sameAs":"https://www.wikidata.org/wiki/Q3130"},
			{"@type":"City","@id":"https://x.test/#sydney2","name":"Sydney","sameAs":"https://www.wikidata.org/wiki/Q3130"}]}`, nil},
		{"no node with an @id names the item", `{"@context":"https://schema.org","@graph":[
			{"@type":"City","name":"Sydney","sameAs":"https://www.wikidata.org/wiki/Q3130"},
			{"@type":"City","name":"Sydney","sameAs":"https://www.wikidata.org/wiki/Q3130"}]}`, nil},
		{"no Wikidata item", `{"@type":"City","name":"Sydney"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, graph.RepeatedItem{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
