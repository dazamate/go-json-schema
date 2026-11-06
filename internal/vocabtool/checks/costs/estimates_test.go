package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

func TestEstimates(t *testing.T) {
	ask := ": mark each up as a MonetaryAmount (value, or minValue and maxValue, currency) on the thing whose cost it is, such as a Service's offers priceSpecification or a PropertyValue, when it is what others charge; give the business's own price as an Offer with seller the business"
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"what moves cost", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page estimates costs ($380–$960, $1,150–$2,850)" + ask},
		}},
		// the second estimate's amounts are the first's, which it took
		{"one estimate twice", audittest.Input{Draft: bareDraft, PageText: "A 1-bedroom move costs $380–$960. A typical one costs $380–$960 too.", PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page estimates costs ($380–$960)" + ask},
		}},
		{"a price with no word of cost", audittest.Input{Draft: bareDraft, PageText: "Pianos: $380.", PageURL: costGuideURL}, nil},
		{"the markup carries them", audittest.Input{Draft: markedDraft, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Estimates{}, tc.in, tc.want) })
	}
}

// A salon's price list notes beside a treatment that lashes cost extra; the
// note is not an estimate of the next treatment's price (willowsbeautysalon.co.uk).
func TestEstimatesSkipSurchargeNotes(t *testing.T) {
	list := `<!doctype html><html><body><main><h1>Treatment price list</h1>
<p>Makeup for Brides £60.00 without lashes extra cost and time is added to apply lashes Make-up To Go without lashes £40.00 Makeup for a prom £40.00 without lashes extra cost and time is added to apply lashes Makeup Trial £45.00</p></main></body></html>`
	audittest.Expect(t, costs.Estimates{}, audittest.Input{Draft: bareDraft, PageHTML: list, PageURL: costGuideURL}, nil)
}
