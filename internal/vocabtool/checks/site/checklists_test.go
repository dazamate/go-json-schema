package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestChecklists(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/moving/","name":"Moving"}`
	done := `{"@context":"https://schema.org","@graph":[` + bare + `,{"@type":"HowTo","name":"Your moving checklist","step":[{"@type":"HowToSection","name":"8 weeks before","itemListElement":[{"@type":"HowToStep","name":"Get three estimates"}]}]}]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a checklist in phases", audittest.Input{Draft: bare, PageHTML: countdownPage, PageURL: countdownURL}, []audit.Issue{
			{Path: "html:#countdown", Severity: audit.Warning, Message: `the page's checklist "Your moving checklist" has 3 phases and 4 tasks: add a HowTo (name, step: a HowToSection for each phase with a HowToStep for each task, name and text word for word, a HowToTip for a task's note, a HowToSupply or HowToTool where a task names one, and itemListOrder https://schema.org/ItemListOrderAscending) in the WebPage's hasPart`},
		}},
		{"a HowTo with the first task", audittest.Input{Draft: done, PageHTML: countdownPage, PageURL: countdownURL}, nil},
		{"a noindex page", audittest.Input{Draft: bare, PageHTML: audittest.Noindex(countdownPage), PageURL: countdownURL}, nil},
		{"the page's text alone", audittest.Input{Draft: bare, PageText: "Your moving checklist. 8 weeks before: get three estimates.", PageURL: countdownURL}, nil},
		{"no page", audittest.Input{Draft: bare, PageURL: countdownURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.Checklists{}, tc.in, tc.want) })
	}
}
