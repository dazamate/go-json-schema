package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestQuoteActions(t *testing.T) {
	apps := `{"@type":"WebApplication","name":"Moving estimate","url":"https://movers.test/costs/#calc","potentialAction":{"@type":"QuoteAction","target":{"@type":"EntryPoint","urlTemplate":"https://movers.test/costs/#calc"}}},
		{"@type":"WebApplication","name":"Tip calculator","url":"https://movers.test/costs/#tips"}`
	plain := `{"@type":"WebApplication","name":"Moving estimate","url":"https://movers.test/costs/#calc","potentialAction":{"@type":"QuoteAction","target":"https://movers.test/costs/#calc"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a quote tool with no QuoteAction", audittest.Input{Draft: toolsDraft, PageHTML: toolsPage, PageURL: toolsURL}, []audit.Issue{
			{Path: "html:#calc", Severity: audit.Warning, Message: `the page's quote tool (#calc) has no QuoteAction: add one to the WebApplication's or WebPage's potentialAction, with an EntryPoint target (urlTemplate the page's URL + #calc, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)`},
		}},
		{"an EntryPoint target", audittest.Input{Draft: withNodes(apps), PageHTML: toolsPage, PageURL: toolsURL}, nil},
		{"a target written as text", audittest.Input{Draft: withNodes(plain), PageHTML: toolsPage, PageURL: toolsURL}, nil},
		{"a page with no tool", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a noindex page", audittest.Input{Draft: toolsDraft, PageHTML: audittest.Noindex(toolsPage), PageURL: toolsURL}, nil},
		{"the page's text alone", audittest.Input{Draft: toolsDraft, PageText: "Moving costs. Free moving estimate.", PageURL: toolsURL}, nil},
		{"no page", audittest.Input{Draft: toolsDraft, PageURL: toolsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.QuoteActions{}, tc.in, tc.want) })
	}
}
