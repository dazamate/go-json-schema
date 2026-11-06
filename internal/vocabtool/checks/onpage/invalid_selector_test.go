package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestInvalidSelector(t *testing.T) {
	const draft = `{"@type":"WebPage","hasPart":[{"@type":"WPHeader","cssSelector":"div["}]}`
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"not CSS", draft, audittest.Input{PageHTML: elementsPage}, []audit.Issue{
			{Path: "$.hasPart[0]", Severity: audit.Warning, Message: `cssSelector "div[" is not a valid CSS selector`},
		}},
		{"a valid selector", `{"@type":"WebPage","hasPart":[{"@type":"WPHeader","cssSelector":"#top"}]}`, audittest.Input{PageHTML: elementsPage}, nil},
		// a page search engines skip lists its sections and asks for nothing
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(elementsPage)}, nil},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.InvalidSelector{}, tc.in, tc.want)
		})
	}
}
