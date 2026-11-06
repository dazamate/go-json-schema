package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

func TestRules(t *testing.T) {
	linked := `<!doctype html><html><body><main><h1>Rules</h1><p>See <a href="https://www.ecfr.gov/current/title-49/part-375">Part 375</a>.</p></main></body></html>`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"rules in the copy", audittest.Input{Draft: rulesBare, PageHTML: rulesPage, PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page cites the rules "49 CFR 375.513", "49 CFR 375.403": add a Legislation for each (name, legislationIdentifier, legislationJurisdiction, legislationType, url, legislationLegalForce https://schema.org/InForce, legislationLegalValue https://schema.org/UnofficialLegalValue for an unofficial rendering such as eCFR; a LegislationObject for a rule's own text) and list it as the WebPage's citation, or its about when a section is about it`},
		}},
		{"a rule in the page's text alone", audittest.Input{Draft: rulesBare, PageText: "Your rights. See 49 CFR 375.513 for the weighing rules.", PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page cites the rules "49 CFR 375.513": add a Legislation for each (name, legislationIdentifier, legislationJurisdiction, legislationType, url, legislationLegalForce https://schema.org/InForce, legislationLegalValue https://schema.org/UnofficialLegalValue for an unofficial rendering such as eCFR; a LegislationObject for a rule's own text) and list it as the WebPage's citation, or its about when a section is about it`},
		}},
		{"a link to a site that publishes laws", audittest.Input{Draft: rulesBare, PageHTML: linked, PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page cites the rules "Part 375": add a Legislation for each (name, legislationIdentifier, legislationJurisdiction, legislationType, url, legislationLegalForce https://schema.org/InForce, legislationLegalValue https://schema.org/UnofficialLegalValue for an unofficial rendering such as eCFR; a LegislationObject for a rule's own text) and list it as the WebPage's citation, or its about when a section is about it`},
		}},
		{"the rules marked up", audittest.Input{Draft: rulesMarked, PageHTML: rulesPage, PageURL: rulesURL}, nil},
		{"a noindex page", audittest.Input{Draft: rulesBare, PageHTML: audittest.Noindex(rulesPage), PageURL: rulesURL}, nil},
		{"no page", audittest.Input{Draft: rulesBare, PageURL: rulesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Rules{}, tc.in, tc.want) })
	}
}
