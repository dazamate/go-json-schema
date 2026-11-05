package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

// The pages are trimmed from councils' own sites: Port Phillip's skip bin permit
// (a gov.au site, a heading outside any section), Melbourne's skip permits
// (one section naming another permit first) and Boroondara's fees
// ("Fees" runs into "Permit").
const (
	portPhillip = `<!doctype html><html><body><div class="ss-react"><h1 class="t-heading-1">Skip bin permit</h1><div>A Skip Bin Permit must be obtained by a skip bin company before a bulk rubbish container is placed on a road or Council land.</div></div>
<main><h2>How do I lodge my application?</h2><p>Apply online. Allow 5 business days.</p><h2>What does the permit cost?</h2><p>Application fee - $40.00 Permit fee per day - $40.00</p></main></body></html>`
	melbourne = `<!doctype html><html><body><main><section><h1>Place skips, bins or containers on the road</h1><h2>Skip permits</h2>
<p>You must not place a skip, bin or container on the road until you have been issued with a skip permit. A permit is not required if the skip will be placed in a construction permit zone.</p>
<h2>Eligibility</h2><p>To apply for a skip permit, you must be authorised. Skip bins can use a Reserved Parking Permit zone.</p>
<h2>Fees</h2><p>Application fee $102.50 per application. A skip permit costs $102.50 per day. Apply for a skip permit online.</p></section></main></body></html>`
	boroondara = `<!doctype html><html><body><main><section><h1>Skip Bin Permit</h1><p>Skip bin companies must apply for a Skip Bin Permit to put a skip bin on a road.</p>
<h2>Fees</h2><p>Permit application fee: $31 per application. Skip bin hire fee: $31 per day. A Skip Bin Permit lasts up to 14 days.</p></section></main></body></html>`
	tenancy = `<!doctype html><html><body><main><section><h2>Section 21 notice</h2><p>The tenant can stay in the property if the tenancy agreement specifically permits it. You must apply within 4 months. Fees apply.</p></section></main></body></html>`
)

const permitAsk = `add a GovernmentPermit (name, issuedBy the GovernmentOrganization, issuedThrough the GovernmentService that issues it, validIn the City or State, validFor, offers for its fee, url)`

func TestPermits(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a permit a city issues", audittest.Input{Draft: rulesBare, PageHTML: rulesPage, PageURL: rulesURL}, []audit.Issue{
			{Path: "html:#local-rules", Severity: audit.Warning, Message: `the page explains the permit "Moving Truck Permit" (from www.boston.gov): add a GovernmentPermit (name, issuedBy the GovernmentOrganization, issuedThrough the GovernmentService that issues it, validIn the City or State, validFor, offers for its fee, url)`},
		}},
		{"a council's own permit page with the permit as its heading", audittest.Input{Draft: rulesBare, PageHTML: portPhillip, PageURL: "https://www.portphillip.vic.gov.au/planning-and-building/construction-permits/skip-bin-permit"}, []audit.Issue{
			{Path: "html:body > div > h1", Severity: audit.Warning, Message: `the page explains the permit "Skip bin permit" (from www.portphillip.vic.gov.au): ` + permitAsk},
		}},
		{"the permit a section names most, not the first it names", audittest.Input{Draft: rulesBare, PageHTML: melbourne, PageURL: "https://www.melbourne.vic.gov.au/place-skips-bins-or-containers-road"}, []audit.Issue{
			{Path: "html:body > main > section", Severity: audit.Warning, Message: `the page explains the permit "Skip permits" (from www.melbourne.vic.gov.au): ` + permitAsk},
		}},
		{"a business's page about the same permit", audittest.Input{Draft: rulesBare, PageHTML: boroondara, PageURL: "https://skips.test/permit/"}, nil},
		{"a verb is no permit", audittest.Input{Draft: rulesBare, PageHTML: tenancy, PageURL: "https://www.gov.uk/evicting-tenants"}, nil},
		{"a GovernmentPermit in the draft", audittest.Input{Draft: rulesMarked, PageHTML: rulesPage, PageURL: rulesURL}, nil},
		{"a noindex page", audittest.Input{Draft: rulesBare, PageHTML: audittest.Noindex(rulesPage), PageURL: rulesURL}, nil},
		{"the page's text alone", audittest.Input{Draft: rulesBare, PageText: "Parking the truck. A standard Moving Truck Permit reserves two spaces. Apply 10 business days before.", PageURL: rulesURL}, nil},
		{"no page", audittest.Input{Draft: rulesBare, PageURL: rulesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Permits{}, tc.in, tc.want) })
	}
}
