package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestMissingProperty(t *testing.T) {
	draft := without(
		`"alternateName":"The cost to move a piano, from upright to grand",`, ``,
		`"alternativeHeadline":"Cost to move a piano: upright vs grand",`, ``,
	)
	fires := []audit.Issue{
		{Path: "$.@graph[2]", Severity: audit.Warning, Message: `no alternateName: add one that carries the keyword "cost to move a piano"`},
		{Path: "$.@graph[2].hasPart[0]", Severity: audit.Warning, Message: `no alternativeHeadline: add one that carries the keyword "cost to move a piano"`},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"properties the nodes lack", draft, audittest.Input{Keyword: kw}, fires},
		// the keyword is held to a page search engines skip too
		{"a noindex page", draft, audittest.Input{Keyword: kw, PageHTML: audittest.Noindex(teaPage)}, fires},
		{"a service's description", without(`"disambiguatingDescription":"The cost to move a piano with a crew."`, `"name":"Piano moving"`), audittest.Input{Keyword: kw}, []audit.Issue{
			{Path: "$.@graph[2].hasPart[0].about", Severity: audit.Warning, Message: `no disambiguatingDescription: add one that carries the keyword "cost to move a piano"`},
		}},
		{"every property there", keywordPage, audittest.Input{Keyword: kw}, nil},
		{"no keyword", draft, audittest.Input{}, nil},
		{"a blank keyword", draft, audittest.Input{Keyword: " \t"}, nil},
		{"a keyword with no words", draft, audittest.Input{Keyword: "?!"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.MissingProperty{}, tc.in, tc.want)
		})
	}
}
