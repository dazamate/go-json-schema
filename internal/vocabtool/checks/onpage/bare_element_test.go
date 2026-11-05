package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestBareElement(t *testing.T) {
	const draft = `{"@type":"WebPage","hasPart":[{"@type":"SiteNavigationElement"}]}`
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"three navs and no selector", draft, audittest.Input{PageHTML: elementsPage}, []audit.Issue{
			{Path: "$.hasPart[0]", Severity: audit.Warning, Message: "SiteNavigationElement has no cssSelector and the page has 3 of its sections: give it the cssSelector of the one it marks up"},
		}},
		{"two nodes for one section", `{"@type":"WebPage","hasPart":[{"@type":"WPFooter"},{"@type":"WPFooter"}]}`, audittest.Input{PageHTML: elementsPage}, []audit.Issue{
			{Path: "$.hasPart[0]", Severity: audit.Warning, Message: "WPFooter has no cssSelector and the page has 1 of its sections: give it the cssSelector of the one it marks up"},
			{Path: "$.hasPart[1]", Severity: audit.Warning, Message: "WPFooter has no cssSelector and the page has 1 of its sections: give it the cssSelector of the one it marks up"},
		}},
		{"the only section of its type", `{"@type":"WebPage","hasPart":[{"@type":"WPFooter"}]}`, audittest.Input{PageHTML: elementsPage}, nil},
		{"with a selector", `{"@type":"WebPage","hasPart":[{"@type":"SiteNavigationElement","cssSelector":"#site-nav"}]}`, audittest.Input{PageHTML: elementsPage}, nil},
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(elementsPage)}, nil},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.BareElement{}, tc.in, tc.want)
		})
	}
}
