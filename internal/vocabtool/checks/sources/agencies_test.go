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

// TestAgenciesOtherCountries pins hosts from other governments, and a
// government site's own subdomains (business.gov.au links ablis.business.gov.au).
func TestAgenciesOtherCountries(t *testing.T) {
	page := func(links ...string) string {
		h := `<!doctype html><html><body><main><h1>Guide</h1><p>`
		for _, l := range links {
			h += `See <a href="` + l + `">the page</a>. `
		}
		return h + `</p></main></body></html>`
	}
	issue := func(list string) []audit.Issue {
		return []audit.Issue{{Path: "html:body", Severity: audit.Warning, Message: "the page links the agencies " + list + ": add each as a GovernmentOrganization (name, url) in the WebPage's mentions, or as the publisher of a citation from it"}}
	}
	for _, tc := range []struct {
		name, url, html string
		want            []audit.Issue
	}{
		{"a gov.uk page", rulesURL, page("https://www.gov.uk/tenant-rights"), issue(`"gov.uk" (www.gov.uk)`)},
		{"an nhs.uk page", rulesURL, page("https://www.nhs.uk/conditions/tooth-decay/"), issue(`"nhs.uk" (www.nhs.uk)`)},
		{"a province's site", rulesURL, page("https://www2.gov.bc.ca/gov/content/home"), issue(`"www2.gov.bc.ca" (www2.gov.bc.ca)`)},
		{"canada.ca", rulesURL, page("https://www.canada.ca/en/revenue-agency.html"), issue(`"canada.ca" (www.canada.ca)`)},
		{"a government site's own subdomain", "https://business.gov.au/Registrations/Register-licences-and-permits", page("https://ablis.business.gov.au/", "https://business.gov.au/x"), nil},
		{"another agency from a government site", "https://business.gov.au/Registrations", page("https://www.ato.gov.au/"), issue(`"ato.gov.au" (www.ato.gov.au)`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, sources.Agencies{}, audittest.Input{Draft: rulesBare, PageHTML: tc.html, PageURL: tc.url}, tc.want)
		})
	}
}
