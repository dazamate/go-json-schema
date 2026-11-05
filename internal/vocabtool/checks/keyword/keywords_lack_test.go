package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestKeywordsLack(t *testing.T) {
	draft := without(`"keywords":["cost to move a piano"]`, `"keywords":["piano movers"]`)
	fires := []audit.Issue{
		{Path: "$.@graph[2]", Severity: audit.Warning, Message: `keywords lacks the target keyword "cost to move a piano": put it first`},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"keywords without it", draft, audittest.Input{Keyword: kw}, fires},
		{"no keywords", without(`"keywords":["cost to move a piano"],`, ``), audittest.Input{Keyword: kw}, fires},
		{"a noindex page", draft, audittest.Input{Keyword: kw, PageHTML: audittest.Noindex(teaPage)}, fires},
		{"keywords with it", keywordPage, audittest.Input{Keyword: kw}, nil},
		{"no keyword", draft, audittest.Input{}, nil},
		{"a blank keyword", draft, audittest.Input{Keyword: "\n"}, nil},
		{"a keyword with no words", draft, audittest.Input{Keyword: "?!"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.KeywordsLack{}, tc.in, tc.want)
		})
	}
}
