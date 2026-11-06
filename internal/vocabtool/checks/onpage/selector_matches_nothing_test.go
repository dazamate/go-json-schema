package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestSelectorMatchesNothing(t *testing.T) {
	const draft = `{"@type":"WebPage","hasPart":[{"@type":"Table","cssSelector":"#nothing"}]}`
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"matches nothing", draft, audittest.Input{PageHTML: elementsPage}, []audit.Issue{
			{Path: "$.hasPart[0]", Severity: audit.Warning, Message: `cssSelector "#nothing" matches nothing on the page`},
		}},
		{"matches a section", `{"@type":"WebPage","hasPart":[{"@type":"Table","cssSelector":"table"}]}`, audittest.Input{PageHTML: elementsPage}, nil},
		// an element the page has that is no section of the type is the wrong-type
		// check's business only when it is a section of another type
		{"matches an element that is no section", `{"@type":"WebPage","hasPart":[{"@type":"Table","cssSelector":"h1"}]}`, audittest.Input{PageHTML: elementsPage}, nil},
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(elementsPage)}, nil},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.SelectorMatchesNothing{}, tc.in, tc.want)
		})
	}
}
