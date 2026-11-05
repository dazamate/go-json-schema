package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestSelectorWrongType(t *testing.T) {
	const draft = `{"@type":"WebPage","hasPart":[{"@type":"WPSideBar","cssSelector":"#site-nav"}]}`
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"selects another type", draft, audittest.Input{PageHTML: elementsPage}, []audit.Issue{
			{Path: "$.hasPart[0]", Severity: audit.Warning, Message: `cssSelector "#site-nav" selects a SiteNavigationElement, not a WPSideBar`},
		}},
		{"selects its own type", `{"@type":"WebPage","hasPart":[{"@type":"SiteNavigationElement","cssSelector":"#site-nav"}]}`, audittest.Input{PageHTML: elementsPage}, nil},
		// a selector that also selects a section of its type is not wrong
		{"selects both types", `{"@type":"WebPage","hasPart":[{"@type":"SiteNavigationElement","cssSelector":"nav"}]}`, audittest.Input{PageHTML: elementsPage}, nil},
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(elementsPage)}, nil},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.SelectorWrongType{}, tc.in, tc.want)
		})
	}
}
