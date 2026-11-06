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
		{"camping gear at approximate prices", audittest.Input{Draft: bareDraft, PageURL: "https://camp.test/budget-camping-gear/", PageHTML: `<!doctype html><html lang="en-US"><body><main><h1>Budget Camping Gear for Beginners</h1><article>
<p>Top Budget Pick: Coleman Sundome 2-Person Tent Approximate Price: $75 Why It Works: Coleman Sundome is the gold standard entry tent.</p>
<p>Top Budget Pick: Teton Sports Tracker Ultralight Sleeping Bag Approximate Price: $45 Why It Works: A bag rated for 20F.</p></article></main></body></html>`}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page prices goods (Sundome 2-Person Tent $75, Ultralight Sleeping Bag $45): add a Product for each with an AggregateOffer (lowPrice, highPrice, priceCurrency, and offerCount: how many sellers or sources the range spans)"},
		}},
		{"paint at retail", audittest.Input{Draft: bareDraft, PageURL: "https://build.test/paint-prices-uk/", PageHTML: `<!doctype html><html lang="en-GB"><body><main><h1>Paint Prices UK</h1><article>
<p>A 2.5L tin retail matt emulsion costs £18–£26 at DIY stores, whereas a 10L tub of trade emulsion is £24–£32.</p></article></main></body></html>`}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page prices goods (2.5L tin £18–£26): add a Product for each with an AggregateOffer (lowPrice, highPrice, priceCurrency, and offerCount: how many sellers or sources the range spans)"},
		}},
		{"a drill at RRP", audittest.Input{Draft: bareDraft, PageURL: "https://diy.test/drills/", PageHTML: `<!doctype html><html lang="en-GB"><body><main><h1>Which drill?</h1><article><p>The Bosch cordless drill has an RRP of £89 to £109 depending on the battery.</p></article></main></body></html>`}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page prices goods (Bosch cordless drill £89 to £109): add a Product for each with an AggregateOffer (lowPrice, highPrice, priceCurrency, and offerCount: how many sellers or sources the range spans)"},
		}},
		{"service prices are not goods", audittest.Input{Draft: bareDraft, PageURL: "https://dental.test/fees/", PageHTML: `<!doctype html><html lang="en-GB"><body><main><h1>Fees</h1><article>
<p>Teeth whitening – top up £60 each. A standard check-up retails at £45 per visit. Installation of a new tap sells for £80 including call-out.</p>
<p>Pest control: $350 for each additional room, and the treatment is $120 a session.</p></article></main></body></html>`}, nil},
		{"a service's approximate price", audittest.Input{Draft: bareDraft, PageURL: "https://movers.test/guide/", PageHTML: `<!doctype html><html lang="en-US"><body><main><h1>Moving guide</h1><article>
<p>A local two-bedroom move: approximate price $1,200. End of lease cleaning: approximate price $350. Kitchen renovation approximate price $25,000.</p></article></main></body></html>`}, nil},
		{"a past sale", audittest.Input{Draft: bareDraft, PageURL: "https://homes.test/blog/market/", PageHTML: `<!doctype html><html lang="en-US"><body><main><h1>Market update</h1><article>
<p>The house next door sold for $850,000 in March.</p></article></main></body></html>`}, nil},
		{"goods priced with no sign of sellers", audittest.Input{Draft: bareDraft, PageText: "Our boxes cost $3.", PageURL: costGuideURL}, nil},
		{"the markup carries them", audittest.Input{Draft: markedDraft, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Goods{}, tc.in, tc.want) })
	}
}
