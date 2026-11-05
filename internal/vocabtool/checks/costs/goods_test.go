package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

func TestGoods(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"goods at typical prices", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page prices goods (box $1.82–$2.69, paper $120–$700): add a Product for each with an AggregateOffer (lowPrice, highPrice, priceCurrency, and offerCount: how many sellers or sources the range spans)"},
		}},
		{"goods priced with no sign of sellers", audittest.Input{Draft: bareDraft, PageText: "Our boxes cost $3.", PageURL: costGuideURL}, nil},
		{"the markup carries them", audittest.Input{Draft: markedDraft, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Goods{}, tc.in, tc.want) })
	}
}
