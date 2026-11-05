package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

// payTable is a guide's table of what movers earn. A pay table's amounts
// are wages, which its own check takes, not rates.
const payTable = `<!doctype html><html><body><main><h1>Movers in San Diego</h1>
<section id="pay"><h2>Pay</h2><p>Movers earn $18 to $23 an hour.</p>
<table><caption>Hourly pay for movers, May 2025</caption><tr><th>Typical</th><td>$20.83</td></tr></table></section>
</main></body></html>`

func TestRates(t *testing.T) {
	hourly := []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: "the page prices $40 an hour: mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode HUR) on the Service or Offer it is the rate of"},
	}
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"rates per hour and per mover", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page prices $50–$75 an hour, $4–$15 per mover per hour: mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode HUR, and referenceQuantity for a "per mover" rate) on the Service or Offer it is the rate of`},
		}},
		{"a rate in the page's text", audittest.Input{Draft: bareDraft, PageText: "Movers charge $40 an hour.", PageURL: costGuideURL}, hourly},
		// the second rate's amount is the first's, which it took
		{"one rate twice", audittest.Input{Draft: bareDraft, PageText: "Movers charge $40 an hour, and packers $40 an hour.", PageURL: costGuideURL}, hourly},
		{"wages a pay table took", audittest.Input{Draft: bareDraft, PageHTML: payTable, PageURL: costGuideURL}, nil},
		{"the markup carries them", audittest.Input{Draft: markedDraft, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Rates{}, tc.in, tc.want) })
	}
}
