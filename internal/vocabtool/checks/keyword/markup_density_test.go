package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestMarkupDensity(t *testing.T) {
	fires := []audit.Issue{
		{Path: "$", Severity: audit.Warning, Message: `the keyword "cost to move a piano" fills 47.2% of the markup's words (12 uses in 127 words), under the 60.0% target: work it into the page's names, headlines and descriptions`},
	}
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"under the target", audittest.Input{Keyword: kw, KeywordDensity: 60}, fires},
		{"a noindex page", audittest.Input{Keyword: kw, KeywordDensity: 60, PageHTML: audittest.Noindex(teaPage)}, fires},
		{"over the default target", audittest.Input{Keyword: kw}, nil},
		{"no keyword", audittest.Input{KeywordDensity: 60}, nil},
		{"a blank keyword", audittest.Input{Keyword: " ", KeywordDensity: 60}, nil},
		{"a keyword with no words", audittest.Input{Keyword: "?!", KeywordDensity: 60}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = keywordPage
			audittest.Expect(t, keyword.MarkupDensity{}, tc.in, tc.want)
		})
	}
}
