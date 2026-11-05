package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestFAQName(t *testing.T) {
	draft := without(`"name":"Cost to move a piano FAQ"`, `"name":"Piano moving FAQ"`)
	fires := []audit.Issue{
		{Path: "$.@graph[2].hasPart[1]", Severity: audit.Warning, Message: `the FAQ's name lacks the keyword "cost to move a piano": name it for the keyword, as its heading on the page`},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"a name without it", draft, audittest.Input{Keyword: kw}, fires},
		{"no name", without(`"name":"Cost to move a piano FAQ",`, ``), audittest.Input{Keyword: kw}, fires},
		{"a noindex page", draft, audittest.Input{Keyword: kw, PageHTML: audittest.Noindex(teaPage)}, fires},
		{"a name with it", keywordPage, audittest.Input{Keyword: kw}, nil},
		// the page's own node is held to the properties the page checks name
		{"the FAQ is the page", `{"@context":"https://schema.org","@type":"FAQPage","url":"https://a.example/faq/","name":"Piano moving FAQ","mainEntity":[]}`, audittest.Input{Keyword: kw, PageURL: "https://a.example/faq/"}, nil},
		{"an FAQ that is not the page", `{"@context":"https://schema.org","@type":"FAQPage","url":"https://a.example/faq/","name":"Piano moving FAQ","mainEntity":[]}`, audittest.Input{Keyword: kw, PageURL: "https://a.example/other/"}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `the FAQ's name lacks the keyword "cost to move a piano": name it for the keyword, as its heading on the page`},
		}},
		{"no keyword", draft, audittest.Input{}, nil},
		{"a blank keyword", draft, audittest.Input{Keyword: " "}, nil},
		{"a keyword with no words", draft, audittest.Input{Keyword: "?!"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.FAQName{}, tc.in, tc.want)
		})
	}
}
