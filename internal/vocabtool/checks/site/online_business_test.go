package site_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

// guidePage is a content site's guide: a header with a logo, an article and
// a footer with a toll-free number. It shows no premises.
const guidePage = `<!doctype html><html lang="en-US"><body>
<header><a href="/"><svg aria-label="Movers Guide"></svg></a><nav aria-label="Main"><a href="/guides/">Guides</a></nav></header>
<main><article><h1>What movers cost</h1>
<p>Movers charge by the hour for local moves and by weight for long ones.</p>
</article></main>
<footer><p>Call <a href="tel:+18334990939">(833) 499-0939</a></p><p>© 2026 Movers Guide</p></footer>
</body></html>`

const guideURL = "https://movers.test/what-movers-cost/"

// published is the site and the Organization that publishes it, with org
// standing in for the Organization's own properties.
func published(org string) string {
	return `{"@context":"https://schema.org","@graph":[
	{"@type":"WebSite","@id":"https://movers.test/#website","url":"https://movers.test/","name":"Movers Guide","publisher":{"@id":"https://movers.test/#org"}},
	{"@type":"Organization","@id":"https://movers.test/#org","name":"Movers Guide",` + org + `}]}`
}

// inFooter is the guide with a line put first in its footer.
func inFooter(line string) string {
	return strings.Replace(guidePage, "<footer>", "<footer><p>"+line+"</p>", 1)
}

func TestOnlineBusiness(t *testing.T) {
	draft := published(`"url":"https://movers.test/","telephone":"+18334990939"`)
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a publisher with no premises", audittest.Input{Draft: draft, PageHTML: guidePage, PageURL: guideURL}, []audit.Issue{
			{Path: "$.@graph[1]", Severity: audit.Warning, Message: `Organization "Movers Guide" shows no premises and works through its website: type it OnlineBusiness`},
		}},
		// a street address or opening hours anywhere on the page, its footer
		// included, are premises
		{"a street address in the footer", audittest.Input{Draft: draft, PageHTML: inFooter("12 Harbour Street, Sydney NSW 2000"), PageURL: guideURL}, nil},
		{"opening hours in the footer", audittest.Input{Draft: draft, PageHTML: inFooter("Mon–Fri 8am–5pm"), PageURL: guideURL}, nil},
		{"an address in the markup", audittest.Input{Draft: published(`"url":"https://movers.test/","address":{"@type":"PostalAddress","addressCountry":"US"}`), PageHTML: guidePage, PageURL: guideURL}, nil},
		// a trade that works at its clients' premises is no online business
		{"areas covered", audittest.Input{Draft: draft, PageHTML: inFooter("Prices shown exclude Rural charges that may apply for addresses outside the urban areas covered – please enquire."), PageURL: guideURL}, nil},
		{"areas we cover", audittest.Input{Draft: draft, PageHTML: inFooter("Areas We Cover West London & Surrey"), PageURL: guideURL}, nil},
		{"a service area", audittest.Input{Draft: draft, PageHTML: inFooter("Our service area is Auckland and Waikato."), PageURL: guideURL}, nil},
		{"a call out fee", audittest.Input{Draft: draft, PageHTML: inFooter("$140 for a call out fee or for false alerts"), PageURL: guideURL}, nil},
		{"we come to you", audittest.Input{Draft: draft, PageHTML: inFooter("We come to you, whenever suits."), PageURL: guideURL}, nil},
		{"a guide that says movers come to you", audittest.Input{Draft: draft, PageHTML: inFooter("Most movers come to your home to quote and the areas they cover vary."), PageURL: guideURL}, []audit.Issue{
			{Path: "$.@graph[1]", Severity: audit.Warning, Message: `Organization "Movers Guide" shows no premises and works through its website: type it OnlineBusiness`},
		}},
		{"no url", audittest.Input{Draft: published(`"telephone":"+18334990939"`), PageHTML: guidePage, PageURL: guideURL}, nil},
		// an Organization the page only mentions is not the business
		{"a mentioned agency", audittest.Input{Draft: `{"@context":"https://schema.org","@graph":[
			{"@type":"WebSite","url":"https://movers.test/","name":"Movers Guide","publisher":{"@type":"Person","name":"Sam Lee"}},
			{"@type":"Organization","name":"FMCSA","url":"https://www.fmcsa.dot.gov/"}]}`, PageHTML: guidePage, PageURL: guideURL}, nil},
		// its Wikidata class told it first
		{"a node another signal told", audittest.Input{Draft: published(`"url":"https://movers.test/","additionalType":"https://www.wikidata.org/wiki/Q3918"`), PageHTML: guidePage, PageURL: guideURL}, nil},
		{"a shop's page", audittest.Input{Draft: draft, PageHTML: audittest.Selling(guidePage), PageURL: guideURL}, nil},
		{"a noindex page", audittest.Input{Draft: draft, PageHTML: audittest.Noindex(guidePage), PageURL: guideURL}, nil},
		// the text shows no premises either, so only the missing document
		// keeps this one quiet
		{"the page's text alone", audittest.Input{Draft: draft, PageText: "What movers cost. Movers charge by the hour.", PageURL: guideURL}, nil},
		{"no page", audittest.Input{Draft: draft, PageURL: guideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.OnlineBusiness{}, tc.in, tc.want) })
	}
}

// A government's or university's site works through its website too, but it
// is no online business (www.gov.uk).
func TestOnlineBusinessSkipsPublicBodies(t *testing.T) {
	gov := `{"@context":"https://schema.org","@graph":[
{"@type":"WebSite","@id":"https://www.gov.uk/#website","url":"https://www.gov.uk/","name":"GOV.UK","publisher":{"@id":"https://www.gov.uk/#org"}},
{"@type":"Organization","@id":"https://www.gov.uk/#org","name":"GOV.UK","url":"https://www.gov.uk/"}]}`
	audittest.Expect(t, site.OnlineBusiness{}, audittest.Input{Draft: gov, PageHTML: guidePage, PageURL: "https://www.gov.uk/evicting-tenants/section-21-and-section-8-notices"}, nil)
	uni := strings.NewReplacer("www.gov.uk", "www.ox.ac.uk", "GOV.UK", "University of Oxford").Replace(gov)
	audittest.Expect(t, site.OnlineBusiness{}, audittest.Input{Draft: uni, PageHTML: guidePage, PageURL: "https://www.ox.ac.uk/admissions"}, nil)
}
