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
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Estimates{}, tc.in, tc.want) })
	}
}
