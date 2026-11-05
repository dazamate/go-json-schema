package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestDeadAnchor(t *testing.T) {
	const pageURL = "https://x.test/tea/"
	draft := func(link string) string {
		return `{"@type":"WebPage","url":"` + pageURL + `","hasPart":[{"@type":"WebPageElement","url":"` + link + `","sameAs":"` + pageURL + `#gone"}]}`
	}
	fires := []audit.Issue{
		{Path: "$.hasPart[0].url", Severity: audit.Warning, Message: "https://x.test/tea/#step-2 points at #step-2, which is no id on the page"},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"an id the page lacks", draft(pageURL + "#step-2"), audittest.Input{PageHTML: teaPage, PageURL: pageURL}, fires},
		{"a noindex page", draft(pageURL + "#step-2"), audittest.Input{PageHTML: audittest.Noindex(teaPage), PageURL: pageURL}, fires},
		{"an id the page has", draft(pageURL + "#step-1"), audittest.Input{PageHTML: teaPage, PageURL: pageURL}, nil},
		{"another page", draft("https://x.test/other/#step-2"), audittest.Input{PageHTML: teaPage, PageURL: pageURL}, nil},
		{"the page's text alone", draft(pageURL + "#step-2"), audittest.Input{PageText: "Tea ceremony.", PageURL: pageURL}, nil},
		{"no address", draft(pageURL + "#step-2"), audittest.Input{PageHTML: teaPage}, nil},
		{"no page", draft(pageURL + "#step-2"), audittest.Input{PageURL: pageURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.DeadAnchor{}, tc.in, tc.want)
		})
	}
}
