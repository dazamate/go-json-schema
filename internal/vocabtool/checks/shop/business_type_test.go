package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

// storePage is the shop's page with its street address, and no cart link.
const storePage = `<!doctype html><html><head><title>Visit us · Harbour Jewellers</title></head><body>
<header><a href="/">Harbour Jewellers</a></header>
<main><h1>Visit us</h1><p>Find us at 12 Harbour Street, Sydney NSW 2000.</p></main>
<footer><p>© 2026 Harbour Jewellers</p></footer></body></html>`

const (
	storeURL  = "https://harbour.test/visit/"
	storeText = "Visit us. Find us at 12 Harbour Street, Sydney NSW 2000."
	// guidePage is a guide on a site that sells nothing and shows no address.
	guidePage = `<!doctype html><html lang="en-US"><body><main><article><h1>What movers cost</h1>
<p>Movers charge by the hour for local moves and by weight for long ones.</p></article></main></body></html>`
	guideURL = "https://movers.test/what-movers-cost/"
	// business is the shop's business, typed no more specifically than an
	// Organization.
	business = `{"@context":"https://schema.org","@type":"Organization","name":"Harbour Jewellers","url":"https://harbour.test/"}`
)

func TestBusinessType(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a shop that sells online", audittest.Input{Draft: business, PageHTML: aboutPage, PageURL: aboutURL}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "the site sells online: type this Organization OnlineStore"},
		}},
		{"a shop with a street address", audittest.Input{Draft: business, PageHTML: storePage, PageURL: storeURL}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `the page shows the street address "12 Harbour Street, Sydney NSW 2000": type this Organization JewelryStore`},
		}},
		{"typed for the shop", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"OnlineStore","name":"Harbour Jewellers","url":"https://harbour.test/"}`, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		// its Wikidata class told it first
		{"a node another signal told", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Organization","name":"Harbour Jewellers","url":"https://harbour.test/","additionalType":"https://www.wikidata.org/wiki/Q3918"}`, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a site that neither sells nor shows an address", audittest.Input{Draft: business, PageHTML: guidePage, PageURL: guideURL}, nil},
		{"a noindex page", audittest.Input{Draft: business, PageHTML: audittest.Noindex(storePage), PageURL: storeURL}, nil},
		// the text shows the address, so only the missing document keeps this
		// one quiet
		{"the page's text alone", audittest.Input{Draft: business, PageText: storeText, PageURL: storeURL}, nil},
		{"no page", audittest.Input{Draft: business, PageURL: storeURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.BusinessType{}, tc.in, tc.want) })
	}
}
