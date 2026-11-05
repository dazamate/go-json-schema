package keyword_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestDescriptionDensity(t *testing.T) {
	long := "The cost to move a piano " + strings.Repeat("depends on the stairs and the distance ", 20) // 1 use in 146 words
	draft := without(`"Check the cost to move a piano before you book."`, `"`+long+`"`)
	fires := []audit.Issue{
		{Path: "$.@graph[2].hasPart[2].description", Severity: audit.Warning, Message: `description gives the keyword "cost to move a piano" 3.4% of its words (1 use in 146 words), under the 5.0% target: use it more, or say less`},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"under the target", draft, audittest.Input{Keyword: kw, KeywordDensity: 5}, fires},
		{"a noindex page", draft, audittest.Input{Keyword: kw, KeywordDensity: 5, PageHTML: audittest.Noindex(teaPage)}, fires},
		// 5 words of 146 is 3.4%, over the 3% default
		{"over the default target", draft, audittest.Input{Keyword: kw}, nil},
		{"a description without it is another check's", without(`"description":"Check the cost to move a piano before you book."`, `"description":"Five steps."`), audittest.Input{Keyword: kw, KeywordDensity: 5}, nil},
		{"no keyword", draft, audittest.Input{KeywordDensity: 5}, nil},
		{"a blank keyword", draft, audittest.Input{Keyword: "  ", KeywordDensity: 5}, nil},
		{"a keyword with no words", draft, audittest.Input{Keyword: "?!", KeywordDensity: 5}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.DescriptionDensity{}, tc.in, tc.want)
		})
	}
}
