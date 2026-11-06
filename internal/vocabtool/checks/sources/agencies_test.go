package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

func TestAgencies(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"two agencies linked", audittest.Input{Draft: rulesBare, PageHTML: rulesPage, PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the agencies "Federal Motor Carrier Safety Administration" (www.fmcsa.dot.gov), "boston.gov" (www.boston.gov): add each as a GovernmentOrganization (name, url) in the WebPage's mentions, or as the publisher of a citation from it`},
		}},
		{"the agencies marked up", audittest.Input{Draft: rulesMarked, PageHTML: rulesPage, PageURL: rulesURL}, nil},
		{"a noindex page", audittest.Input{Draft: rulesBare, PageHTML: audittest.Noindex(rulesPage), PageURL: rulesURL}, nil},
		{"the page's text alone", audittest.Input{Draft: rulesBare, PageText: "More on the rules from the Federal Motor Carrier Safety Administration.", PageURL: rulesURL}, nil},
		{"no page", audittest.Input{Draft: rulesBare, PageURL: rulesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Agencies{}, tc.in, tc.want) })
	}
}
