package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

const rulesAsk = `add a Legislation for each (name, legislationIdentifier, legislationJurisdiction, legislationType, url, legislationLegalForce https://schema.org/InForce, legislationLegalValue https://schema.org/UnofficialLegalValue for an unofficial rendering such as eCFR; a LegislationObject for a rule's own text) and list it as the WebPage's citation, or its about when a section is about it`

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
		{"Acts named in a landlord guide", audittest.Input{Draft: rulesBare, PageText: "What is a Section 21 notice? A Section 21 notice, provided under section 21 of the Housing Act 1988, ends an assured shorthold tenancy. The Renters’ Rights Act 2025 abolishes it. Section 2 of this guide covers costs.", PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page cites the rules "Housing Act 1988", "Renters’ Rights Act 2025": ` + rulesAsk},
		}},
		{"an Act named with a comma before its year", audittest.Input{Draft: rulesBare, PageText: "Some employees are exempt from the overtime provisions of the Employment Standards Act, 2000 (ESA).", PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page cites the rules "Employment Standards Act, 2000": ` + rulesAsk},
		}},
		{"a heading's words before an Act", audittest.Input{Draft: rulesBare, PageText: "Timeline The Renters Rights Act 2025 Related Guides. Landlords must follow the Renters Rights Act 2025 from May.", PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page cites the rules "Renters Rights Act 2025": ` + rulesAsk},
		}},
		{"an Act the draft names", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/boston/","citation":{"@type":"Legislation","name":"Housing Act 1988"}}`, PageText: "A notice under section 21 of the Housing Act 1988 ends the tenancy.", PageURL: rulesURL}, nil},
		{"a link to a state's legislation site", audittest.Input{Draft: rulesBare, PageHTML: `<!doctype html><html><body><main><h1>Rent</h1><p>See the <a href="https://legislation.nsw.gov.au/view/html/inforce/current/act-2010-042">Residential Tenancies Act 2010</a>.</p></main></body></html>`, PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page cites the rules "Residential Tenancies Act 2010": ` + rulesAsk},
		}},
		{"a shop's House Rules and Pre-Order dates", audittest.Input{Draft: rulesBare, PageText: "House Rules 2026 apply. Pre-Order 2026 stock ships in May. Act 2 of the show starts at 8.", PageURL: rulesURL}, nil},
		{"the rules marked up", audittest.Input{Draft: rulesMarked, PageHTML: rulesPage, PageURL: rulesURL}, nil},
		{"a noindex page", audittest.Input{Draft: rulesBare, PageHTML: audittest.Noindex(rulesPage), PageURL: rulesURL}, nil},
		{"no page", audittest.Input{Draft: rulesBare, PageURL: rulesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Rules{}, tc.in, tc.want) })
	}
}
