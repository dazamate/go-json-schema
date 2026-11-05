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

// TestRatesOtherTrades pins rates worded as other trades word them: the
// tutor's "$10 per hour" and a dentist's "$125/appt", a pest controller's
// "$140 per nest", a SaaS plan's "$10 per seat / month", an Indian shop's
// "(₹53/day)", a landscaper's "$45 per m²" and a gym's "£24.99 a month". Plain
// copy that only looks like one stays silent.
func TestRatesOtherTrades(t *testing.T) {
	msg := func(list, codes string) []audit.Issue {
		return []audit.Issue{{Path: "html:body", Severity: audit.Warning, Message: "the page prices " + list + ": mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode " + codes + ") on the Service or Offer it is the rate of"}}
	}
	for _, tc := range []struct {
		name, text string
		want       []audit.Issue
	}{
		{"a tutor's hour", "Min Rate $10 per hour $20 per hour", msg("$10 per hour, $20 per hour", "HUR")},
		{"a dentist's appointment", "Nitrous $125/appt Certain PPO plans may cover up to 100%", msg("$125/appt", "C62 with unitText appt")},
		{"a unit the list does not know", "Bumblebees nest relocation- $140 per nest. Extraction per tooth £149.00 Per Additional tooth £129.00", nil},
		{"a gym's month", "Join for £24.99 a month with no contract.", msg("£24.99 a month", "MON")},
		{"a rate a year", "Members pay $40 a year or $4 per month.", msg("$40 a year, $4 per month", "ANN or MON")},
		{"an area", "Turf costs $45 per m² laid, or $50/sqm for sloping sites.", msg("$45 per m², $50/sqm", "MTK")},
		{"a per day price in rupees", "Extended warranty ₹19,249.00 (₹53/day) MRP", msg("₹53/day", "DAY")},
		{"a SaaS seat", "Plus $10 per seat / month for small teams.", []audit.Issue{{Path: "html:body", Severity: audit.Warning, Message: `the page prices $10 per seat / month: mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode MON, and referenceQuantity for a "per seat" rate) on the Service or Offer it is the rate of`}}},
		{"a percentage", "The fee is $20 per cent of the bill.", nil},
		{"a price then a word", "It costs $20 a few times a year, per the contract.", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, costs.Rates{}, audittest.Input{Draft: bareDraft, PageText: tc.text, PageURL: costGuideURL}, tc.want)
		})
	}
}
