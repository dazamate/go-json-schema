package costs_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

// truckArticle breaks a truck's cost into priced parts beside its total.
const truckArticle = `<article><header><h3>Rent a truck</h3></header><p><output><data value="103">$58–$103</data></output></p>
<dl><div><dt>A 20 ft truck, one day in town</dt><dd>$39–$40</dd></div><div><dt>Mileage, 20 miles round trip</dt><dd>$13–$50</dd></div><div><dt>Gasoline for 20 miles</dt><dd>$6–$13</dd></div><div><dt>Who lifts</dt><dd>The household</dd></div></dl></article>`

// costGuide is a content site's cost guide: it estimates what other
// businesses charge, in rates, breakdowns, ranges and goods prices. It sells
// none of them.
const costGuide = `<!doctype html><html lang="en-US"><body><main><h1>Cost to Move a Piano, From Upright to Grand</h1>
<article><p>Moving an upright piano across town costs $380–$960, and a grand $1,150–$2,850. Each mover adds $50–$75 an hour, and long carries run $4–$15 per mover per hour.</p>
<p>A medium box costs $1.82–$2.69 each, and boxes, tape and paper for the rooms listed come to $120–$700, by typical published prices.</p>
<table><tr><th>Piano</th><th>Price</th></tr><tr><td>Upright</td><td>$300</td></tr><tr><td>Grand</td><td>$900</td></tr></table></article>
` + truckArticle + `
</main></body></html>`

const (
	costGuideURL = "https://movers.test/cost-to-move-a-piano/"
	bareDraft    = `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/cost-to-move-a-piano/","name":"Piano moving cost"}`
	// markedDraft carries every price the cost guide shows, each as the
	// type that fits it
	markedDraft = `{"@context":"https://schema.org","@graph":[` + bareDraft + `,
		{"@type":"Service","name":"Piano moving","offers":{"@type":"Offer","priceSpecification":[
			{"@type":"UnitPriceSpecification","minPrice":50,"maxPrice":75,"priceCurrency":"USD","unitCode":"HUR"},
			{"@type":"UnitPriceSpecification","minPrice":4,"maxPrice":15,"priceCurrency":"USD","unitCode":"HUR"},
			{"@type":"CompoundPriceSpecification","minPrice":58,"maxPrice":103,"priceCurrency":"USD","priceComponent":[
				{"@type":"UnitPriceSpecification","minPrice":39,"maxPrice":40,"priceCurrency":"USD"},
				{"@type":"UnitPriceSpecification","minPrice":13,"maxPrice":50,"priceCurrency":"USD","priceComponentType":"https://schema.org/DistanceFee"},
				{"@type":"UnitPriceSpecification","minPrice":6,"maxPrice":13,"priceCurrency":"USD"}]}]}},
		{"@type":"MonetaryAmount","minValue":380,"maxValue":960,"currency":"USD"},
		{"@type":"MonetaryAmount","minValue":1150,"maxValue":2850,"currency":"USD"},
		{"@type":"MonetaryAmount","minValue":120,"maxValue":700,"currency":"USD"},
		{"@type":"MonetaryAmount","value":300,"currency":"USD"},{"@type":"MonetaryAmount","value":900,"currency":"USD"},
		{"@type":"Product","name":"Medium box","offers":{"@type":"AggregateOffer","lowPrice":1.82,"highPrice":2.69,"priceCurrency":"USD"}}]}`
)

// selling is the page with a link to a cart: a shop's page.
func selling(page string) string {
	return strings.Replace(page, "<body>", `<body><header><nav><a href="/cart">Cart</a></nav></header>`, 1)
}

// noindex is the page kept out of search.
func noindex(page string) string {
	if strings.Contains(page, "<head>") {
		return strings.Replace(page, "<head>", `<head><meta name="robots" content="noindex">`, 1)
	}
	return strings.Replace(page, "<body>", `<head><meta name="robots" content="noindex"></head><body>`, 1)
}

func TestBreakdowns(t *testing.T) {
	broken := []audit.Issue{
		{Path: "html:body > main > article:nth-of-type(2)", Severity: audit.Warning, Message: `the page breaks "Rent a truck" ($58–$103) into 3 priced parts: mark it up as a CompoundPriceSpecification (the PriceSpecification that sums its parts: minPrice, maxPrice, priceCurrency) with each part a priceComponent (a UnitPriceSpecification with priceComponentType such as DistanceFee for mileage, or a DeliveryChargeSpecification for a delivery or pickup charge)`},
	}
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a total broken into priced parts", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: costGuideURL}, broken},
		// the second breakdown's amounts are the first's, which it took
		{"the same breakdown twice", audittest.Input{Draft: bareDraft, PageHTML: strings.Replace(costGuide, truckArticle, truckArticle+"\n"+truckArticle, 1), PageURL: costGuideURL}, broken},
		{"the markup carries it", audittest.Input{Draft: markedDraft, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"only the page's text", audittest.Input{Draft: bareDraft, PageText: "Rent a truck: $58–$103. A 20 ft truck, one day in town: $39–$40. Mileage, 20 miles round trip: $13–$50.", PageURL: costGuideURL}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Breakdowns{}, tc.in, tc.want) })
	}
}
