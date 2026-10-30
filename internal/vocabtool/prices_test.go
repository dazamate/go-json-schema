package vocabtool

import (
	"strings"
	"testing"
)

// costGuide is a content site's cost guide: it estimates what other
// businesses charge, in rates, breakdowns, ranges and goods prices. It sells
// none of them.
const costGuide = `<!doctype html><html lang="en-US"><body><main><h1>Cost to Move a Piano, From Upright to Grand</h1>
<article><p>Moving an upright piano across town costs $380–$960, and a grand $1,150–$2,850. Each mover adds $50–$75 an hour, and long carries run $4–$15 per mover per hour.</p>
<p>A medium box costs $1.82–$2.69 each, and boxes, tape and paper for the rooms listed come to $120–$700, by typical published prices.</p>
<table><tr><th>Piano</th><th>Price</th></tr><tr><td>Upright</td><td>$300</td></tr><tr><td>Grand</td><td>$900</td></tr></table></article>
<article><header><h3>Rent a truck</h3></header><p><output><data value="103">$58–$103</data></output></p>
<dl><div><dt>A 20 ft truck, one day in town</dt><dd>$39–$40</dd></div><div><dt>Mileage, 20 miles round trip</dt><dd>$13–$50</dd></div><div><dt>Gasoline for 20 miles</dt><dd>$6–$13</dd></div><div><dt>Who lifts</dt><dd>The household</dd></div></dl></article>
</main></body></html>`

const costGuideURL = "https://movers.test/cost-to-move-a-piano/"

func TestCheckNamesThePriceTypeForWhatAPriceIs(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/cost-to-move-a-piano/","name":"Piano moving cost"}`
	r := checkSite(t, bare, costGuide, costGuideURL)
	expectWarnings(t, r, [][]string{
		{"$50–$75 an hour", "as a UnitPriceSpecification", "unitCode HUR", "referenceQuantity"},
		{`breaks "Rent a truck" ($58–$103) into 3 priced parts`, "as a CompoundPriceSpecification", "DistanceFee", "DeliveryChargeSpecification", "PriceSpecification that"},
		{"($380–$960, $1,150–$2,850)", "as a MonetaryAmount", "when it is what others charge", "an Offer with seller the business"},
		{"$1.82–$2.69", "add a Product", "AggregateOffer"},
		{"$300", "that no price property carries", "MonetaryAmount"},
		{`about the cost of the service "Move a Piano"`, "no Service describes it"},
	})
	for _, i := range r.Issues {
		if strings.Contains(i.Message, "as an Offer's price or, for a range, an AggregateOffer's") {
			t.Errorf("a cost guide's estimates pushed into an Offer: %s", i.Message)
		}
	}
	// each price is reported once at most, by the ask that fits it
	for _, p := range []string{"$39", "$50–$75", "$380", "$1.82", "$58", "$300"} {
		if n := issueCount(r, p); n > 1 {
			t.Errorf("%s is in %d warnings, want 1", p, n)
		}
	}
	// the markup that carries them leaves nothing to ask
	marked := `{"@context":"https://schema.org","@graph":[` + bare + `,
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
	r = checkSite(t, marked, costGuide, costGuideURL)
	for _, i := range r.Issues {
		for _, w := range []string{"UnitPriceSpecification", "CompoundPriceSpecification", "as a MonetaryAmount", "add a Product", "no price property", "no Service"} {
			if strings.Contains(i.Message, w) {
				t.Errorf("asked with the prices marked up: %s", i.Message)
			}
		}
	}
}

func TestCheckKeepsOffersForAShopsPrices(t *testing.T) {
	r := checkShop(t, "qod-category", "")
	for _, w := range []string{"as a UnitPriceSpecification", "as a CompoundPriceSpecification", "what others charge", "no Service describes it"} {
		if n := issueCount(r, w); n != 0 {
			t.Errorf("a shop's prices got %q: %+v", w, r.Issues)
		}
	}
}
