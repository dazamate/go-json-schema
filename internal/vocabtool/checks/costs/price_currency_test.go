package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

func TestPriceCurrency(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Movers</h1><p>Our movers.</p></main></body></html>`
	const url = "https://movers.test/movers/"
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a price with no currency", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Offer","price":"10"}`, PageHTML: page, PageURL: url}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "this Offer has a price but no priceCurrency: set it (the ISO 4217 code of the currency the page shows: USD, EUR, GBP, AUD…)"},
		}},
		{"a monetary value with no currency", audittest.Input{Draft: `{"@context":"https://schema.org","@graph":[{"@type":"MonetaryAmount","minValue":380,"maxValue":960}]}`, PageHTML: page, PageURL: url}, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: "this MonetaryAmount has a price but no priceCurrency: set it (the ISO 4217 code of the currency the page shows: USD, EUR, GBP, AUD…)"},
		}},
		// the price is real whatever the node is called, so the issue stands, with
		// a neutral noun where there is no type to name
		{"an untyped node with a price", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Service","offers":{"price":"50"}}`, PageHTML: page, PageURL: url}, []audit.Issue{
			{Path: "$.offers", Severity: audit.Warning, Message: "this node has a price but no priceCurrency: set it (the ISO 4217 code of the currency the page shows: USD, EUR, GBP, AUD…)"},
		}},
		{"a priceCurrency", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Offer","price":"10","priceCurrency":"USD"}`, PageHTML: page, PageURL: url}, nil},
		{"a currency", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"MonetaryAmount","value":10,"currency":"USD"}`, PageHTML: page, PageURL: url}, nil},
		{"a value outside a price", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"QuantitativeValue","value":10}`, PageHTML: page, PageURL: url}, nil},
		{"a noindex page", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Offer","price":"10"}`, PageHTML: audittest.Noindex(page), PageURL: url}, nil},
		{"no page", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Offer","price":"10"}`, PageURL: url}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.PriceCurrency{}, tc.in, tc.want) })
	}
}
