package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// productPage is a shop's ring set on sale, with a paid add-on and a unit
// price, each of which its own check reports.
const productPage = `<!doctype html><html><head><link rel="canonical" href="https://harbour.test/products/art-deco-ring-set"></head><body>
<header><a href="/">Harbour Jewellers</a><nav><a href="/cart">Cart</a></nav></header>
<main><h1>Art Deco ring set</h1>
<p><del>$389.00</del> <ins>$320.00</ins> GST included</p>
<form action="/cart/add" method="post"><label><input type="checkbox" name="gift_wrap"> Add gift wrapping for $5.00</label><button>Add to cart</button></form>
<p>Card payments incur a 1.5% surcharge.</p>
<p>Polishing cloth: $4.50 per 100 g of cloth when sold separately.</p></main></body></html>`

// listingPage is a shop's listing of rings: a price filter, a starting
// price and product cards, whose prices the listing check takes.
const listingPage = `<!doctype html><html><head><link rel="canonical" href="https://harbour.test/collections/rings"></head><body>
<header><a href="/">Harbour Jewellers</a><nav><a href="/collections/rings">Rings</a> <a href="/cart">Cart</a></nav></header>
<main><h1>Rings</h1>
<aside aria-label="Filters"><a href="/collections/rings?price=under-100">Under $100</a> <a href="/collections/rings?metal=gold">Gold</a></aside>
<p>48 products. Prices start from $99.00.</p>
<ul><li><a href="/products/opal-ring">Opal ring</a> <del>$189.00</del> <ins>$149.00</ins></li>
<li><a href="/products/garnet-ring">Garnet ring</a> $129.00</li>
<li><a href="/products/pearl-ring">Pearl ring</a> $99.00</li></ul></main>
<footer><p>© 2026 Harbour Jewellers</p></footer></body></html>`

const (
	productURL = "https://harbour.test/products/art-deco-ring-set"
	listingURL = "https://harbour.test/collections/rings"
	noDraft    = `{"@context":"https://schema.org","@graph":[]}`
)

func TestOfferPrices(t *testing.T) {
	priced := `{"@context":"https://schema.org","@type":"Offer","price":320,"priceCurrency":"AUD"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a price no shop check reports", audittest.Input{Draft: noDraft, PageHTML: productPage, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows prices ($320.00) that no price property carries: mark each up on the item it is the price of, as an Offer's price or, for a range, an AggregateOffer's lowPrice and highPrice, with priceCurrency and offerCount"},
		}},
		{"an Offer carries it", audittest.Input{Draft: priced, PageHTML: productPage, PageURL: productURL}, nil},
		{"a listing's filter, cards and starting price", audittest.Input{Draft: noDraft, PageHTML: listingPage, PageURL: listingURL}, nil},
		{"a site that sells nothing", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.OfferPrices{}, tc.in, tc.want) })
	}
}

// offerPricesMsg is the offer-prices issue for the amounts it lists.
func offerPricesMsg(amounts string) audit.Issue {
	return audit.Issue{Path: "html:body", Severity: audit.Warning, Message: "the page shows prices (" + amounts + ") that no price property carries: mark each up on the item it is the price of, as an Offer's price or, for a range, an AggregateOffer's lowPrice and highPrice, with priceCurrency and offerCount"}
}

// TestOfferPricesLedger pins the takes that report nothing: a taker takes
// its amounts whether or not it reports them, so the prices checks stay quiet
// on them; one that does not take an amount leaves it to offer-prices.
func TestOfferPricesLedger(t *testing.T) {
	const freeOver = `<!doctype html><html><head><link rel="canonical" href="https://harbour.test/products/art-deco-ring-set"></head><body>
<header><a href="/">Harbour Jewellers</a><nav><a href="/cart">Cart</a></nav></header>
<main><h1>Art Deco ring set</h1>
<p><ins>$320.00</ins> GST included</p>
<p>Free shipping on orders over $60.</p></main></body></html>`
	t.Run("a threshold the draft holds bare is still taken", func(t *testing.T) {
		// the draft's orderValue 60 is the threshold, so shipping-rate says nothing
		// of it, yet it still takes the amount
		in := audittest.Input{Draft: `{"@context":"https://schema.org","@type":"ShippingConditions","orderValue":60}`, PageHTML: freeOver, PageURL: productURL}
		audittest.Expect(t, policies.ShippingRate{}, in, nil)
		audittest.Expect(t, costs.OfferPrices{}, in, []audit.Issue{offerPricesMsg("$320.00")})
	})
	t.Run("a threshold nothing carries is taken by shipping-rate", func(t *testing.T) {
		in := audittest.Input{Draft: noDraft, PageHTML: freeOver, PageURL: productURL}
		audittest.Expect(t, costs.OfferPrices{}, in, []audit.Issue{offerPricesMsg("$320.00")})
	})
	t.Run("a unit price the draft gives a referenceQuantity is not taken", func(t *testing.T) {
		// unit-price leaves the price to the draft that has its referenceQuantity,
		// and that draft carries no price, so offer-prices reports it
		in := audittest.Input{Draft: `{"@context":"https://schema.org","@type":"UnitPriceSpecification","referenceQuantity":{"@type":"QuantitativeValue","value":100,"unitCode":"GRM"}}`, PageHTML: productPage, PageURL: productURL}
		audittest.Expect(t, product.UnitPrice{}, in, nil)
		audittest.Expect(t, costs.OfferPrices{}, in, []audit.Issue{offerPricesMsg("$320.00, $4.50")})
	})
}
