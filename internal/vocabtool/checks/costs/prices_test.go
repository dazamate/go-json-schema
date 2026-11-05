package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

func TestPrices(t *testing.T) {
	ask := " that no price property carries: mark each up on the thing it is the price of, as a MonetaryAmount (value, or minValue and maxValue, currency) for a cost or estimate of what others charge, or an Offer's price only for what the business itself sells"
	// a was price on a site that sells nothing is no shop's sale
	struck := `<!doctype html><html><body><main><h1>Movers</h1><p>Movers once charged <del>$50</del> and now charge less.</p></main></body></html>`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"prices no other check reports", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows prices ($300, $900)" + ask},
		}},
		{"a was price on a site that sells nothing", audittest.Input{Draft: bareDraft, PageHTML: struck, PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows prices ($50)" + ask},
		}},
		{"the markup carries them", audittest.Input{Draft: markedDraft, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Prices{}, tc.in, tc.want) })
	}
}

// A price another check reports is left out of prices: the rate check takes
// $40 an hour.
func TestPricesSkipsTaken(t *testing.T) {
	in := audittest.Input{Draft: bareDraft, PageText: "Movers charge $40 an hour.", PageURL: costGuideURL}
	audittest.Expect(t, costs.Rates{}, in, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: "the page prices $40 an hour: mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode HUR) on the Service or Offer it is the rate of"},
	})
	audittest.Expect(t, costs.Prices{}, in, nil)
}
