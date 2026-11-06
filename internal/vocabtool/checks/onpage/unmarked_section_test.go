package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestUnmarkedSection(t *testing.T) {
	const draft = `{"@type":"WebPage","name":"Tea"}`
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"an unmarked footer", draft, audittest.Input{PageHTML: footerPage}, []audit.Issue{
			{Path: "html:body > footer", Severity: audit.Warning, Message: `the page's <footer> has no WPFooter: add {"@type": "WPFooter", "cssSelector": "body > footer"} to the WebPage's hasPart`},
		}},
		{"marked by selector", `{"@type":"WebPage","hasPart":[{"@type":"WPFooter","cssSelector":"footer"}]}`, audittest.Input{PageHTML: footerPage}, nil},
		{"marked as the only one", `{"@type":"WebPage","hasPart":[{"@type":"WPFooter"}]}`, audittest.Input{PageHTML: footerPage}, nil},
		// the breadcrumb is the next check's business
		{"only a breadcrumb", draft, audittest.Input{PageHTML: crumbPage}, nil},
		{"no sections", draft, audittest.Input{PageHTML: teaPage}, nil},
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(footerPage)}, nil},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.UnmarkedSection{}, tc.in, tc.want)
		})
	}
}
