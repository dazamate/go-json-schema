package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
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
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Rates{}, tc.in, tc.want) })
	}
}

// A pay table's wages are taken whether or not the draft carries them: with
// an estimatedSalary in the draft the pay check says nothing, yet the rates
// and prices checks still leave the table's amounts alone.
func TestPayTableTakenWhenCarried(t *testing.T) {
	in := audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Occupation","name":"Mover","estimatedSalary":{"@type":"MonetaryAmountDistribution","name":"base","currency":"USD","duration":"P1H","median":20.83}}`, PageHTML: payTable, PageURL: costGuideURL}
	audittest.Expect(t, sources.PayPercentiles{}, in, nil)
	audittest.Expect(t, costs.Rates{}, in, nil)
	audittest.Expect(t, costs.Prices{}, in, nil)
}
