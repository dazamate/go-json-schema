package shop_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

// The product cards of a shop's listing of rings.
const (
	opalCard   = `<li><a href="/products/opal-ring">Opal ring</a> <del>$189.00</del> <ins>$149.00</ins></li>`
	garnetCard = `<li><a href="/products/garnet-ring">Garnet ring</a> $129.00</li>`
	pearlCard  = `<li><a href="/products/pearl-ring">Pearl ring</a> $99.00</li>`
)

// listingPage is a shop's listing of rings: a filter with a price in it, a
// product count, a starting price, a sort control and the product cards.
func listingPage(cards ...string) string {
	return `<!doctype html><html><head><link rel="canonical" href="https://harbour.test/collections/rings"></head><body>
<header><a href="/">Harbour Jewellers</a><nav><a href="/collections/rings">Rings</a> <a href="/cart">Cart</a></nav></header>
<main><h1>Rings</h1>
<aside aria-label="Filters"><a href="/collections/rings?price=under-100">Under $100</a> <a href="/collections/rings?metal=gold">Gold</a></aside>
<p>48 products. Prices start from $99.00.</p>
<select id="sort" name="sort_by"><option value="manual">Featured</option><option value="price-ascending" selected>Price, low to high</option></select>
<ul>` + strings.Join(cards, "\n") + `</ul></main>
<footer><p>© 2026 Harbour Jewellers</p></footer></body></html>`
}

var listing = listingPage(opalCard, garnetCard, pearlCard)

const (
	listingURL = "https://harbour.test/collections/rings"
	homeURL    = "https://harbour.test/"
	noDraft    = `{"@context":"https://schema.org","@graph":[]}`
)

func TestListing(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"ItemList","itemListElement":[
		{"@type":"ListItem","position":1,"url":"https://harbour.test/products/opal-ring"},
		{"@type":"ListItem","position":2,"url":"https://harbour.test/products/garnet-ring"},
		{"@type":"ListItem","position":3,"url":"https://harbour.test/products/pearl-ring"}]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a listing's cards", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: listingURL}, []audit.Issue{
			{Path: "html:body > main > ul", Severity: audit.Warning, Message: "the page lists 3 products (Opal ring, Garnet ring, Pearl ring) that no ItemList carries: add an ItemList as the CollectionPage's mainEntity (numberOfItems, itemListOrder), with a ListItem for each product (position, url) holding its Product (name, url, image, offers an Offer with price and priceCurrency)"},
		}},
		{"a row of products on the home page", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: homeURL}, []audit.Issue{
			{Path: "html:body > main > ul", Severity: audit.Warning, Message: "the page lists 3 products (Opal ring, Garnet ring, Pearl ring) that no ItemList carries: add an ItemList as the WebPage's hasPart (name its heading), with a ListItem for each product (position, url) holding its Product (name, url, image, offers an Offer with price and priceCurrency)"},
		}},
		{"an ItemList carries the cards", audittest.Input{Draft: carried, PageHTML: listing, PageURL: listingURL}, nil},
		{"one card is no listing", audittest.Input{Draft: noDraft, PageHTML: listingPage(opalCard), PageURL: listingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(listing), PageURL: listingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: listingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.Listing{}, tc.in, tc.want) })
	}
}
