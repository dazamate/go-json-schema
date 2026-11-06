package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestLacksKeyword(t *testing.T) {
	draft := without(
		`"name":"Cost to Move a Piano: Upright vs Grand"`, `"name":"Moving a piano: upright vs grand"`,
		`"description":"Get a free estimate of the cost to move a piano."`, `"description":"Get a free estimate for your move."`,
		`"description":"Check the cost to move a piano before you book."`, `"description":"Five steps."`,
	)
	fires := []audit.Issue{
		{Path: "$.@graph[2].name", Severity: audit.Warning, Message: `name "Moving a piano: upright vs grand" lacks the keyword "cost to move a piano"`},
		{Path: "$.@graph[2].description", Severity: audit.Warning, Message: `description "Get a free estimate for your move." lacks the keyword "cost to move a piano"`},
		{Path: "$.@graph[2].hasPart[2].description", Severity: audit.Warning, Message: `description "Five steps." lacks the keyword "cost to move a piano"`},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"names and descriptions without it", draft, audittest.Input{Keyword: kw}, fires},
		{"a noindex page", draft, audittest.Input{Keyword: kw, PageHTML: audittest.Noindex(teaPage)}, fires},
		// the FAQ page's name has its own check; entities and data are not held to the keyword
		{"an entity and a data node", keywordPage, audittest.Input{Keyword: kw}, nil},
		{"a node other pages share", without(`"@id":"https://a.example/piano/#steps"`, `"@id":"https://a.example/#calculator"`, `"description":"Check the cost to move a piano before you book."`, `"description":"A free calculator."`), audittest.Input{Keyword: kw}, nil},
		{"no keyword", draft, audittest.Input{}, nil},
		{"a blank keyword", draft, audittest.Input{Keyword: "  "}, nil},
		{"a keyword with no words", draft, audittest.Input{Keyword: "?!"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.LacksKeyword{}, tc.in, tc.want)
		})
	}
}
