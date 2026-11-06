package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestPageName(t *testing.T) {
	const draft = `{"@context":"https://schema.org","@type":"WebPage","description":"Tea."}`
	fires := []audit.Issue{{Path: "$", Severity: audit.Warning, Message: "this WebPage has no url or name: give it the page's"}}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"no url or name", draft, audittest.Input{PageHTML: teaPage}, fires},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, fires},
		// the markup's own faults are held to a page search engines skip too
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"no name", `{"@type":"WebPage","url":"https://x.test/tea/"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "this WebPage has no name: give it the page's"},
		}},
		{"a subtype with no url", `{"@type":"AboutPage","name":"About"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "this AboutPage has no url: give it the page's"},
		}},
		{"a url and a name", `{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea"}`, audittest.Input{PageText: "Tea."}, nil},
		{"no WebPage", `{"@type":"Article","headline":"Tea"}`, audittest.Input{PageText: "Tea."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, graph.PageName{}, tc.in, tc.want)
		})
	}
}
