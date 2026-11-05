package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
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
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.OfferPrices{}, tc.in, tc.want) })
	}
}
