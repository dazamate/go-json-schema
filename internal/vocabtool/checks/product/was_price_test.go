package product_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// The parts of a shop's product page that sell: its cart link and its
// add-to-cart form.
const (
	cartLink = `<nav><a href="/cart">Cart</a></nav>`
	buyForm  = `<form action="/cart/add" method="post"><label><input type="checkbox" name="gift_wrap"> Add gift wrapping for $5.00</label><button>Add to cart</button></form>`
)

// productPage is a ring set on sale, with a paid add-on, a card surcharge
// and a unit price.
const productPage = `<!doctype html><html><head><link rel="canonical" href="https://harbour.test/products/art-deco-ring-set"></head><body>
<header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Art Deco ring set</h1>
<p><del>$389.00</del> <ins>$320.00</ins> GST included</p>
` + buyForm + `
<p>Card payments incur a 1.5% surcharge.</p>
<p>Polishing cloth: $4.50 per 100 g of cloth when sold separately.</p></main></body></html>`

const (
	productURL = "https://harbour.test/products/art-deco-ring-set"
	noDraft    = `{"@context":"https://schema.org","@graph":[]}`
)

// sellingNoMore is the page without its cart link and add-to-cart form.
func sellingNoMore(page string) string {
	return strings.NewReplacer(cartLink, "", buyForm, "").Replace(page)
}

// struckThrough is a draft that carries a was price.
func struckThrough(price string) string {
	return `{"@context":"https://schema.org","@type":"Offer","price":320,"priceCurrency":"AUD","priceSpecification":{"@type":"UnitPriceSpecification","priceType":"https://schema.org/StrikethroughPrice","price":` + price + `,"priceCurrency":"AUD"}}`
}

func TestWasPrice(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a product page's was price", audittest.Input{Draft: noDraft, PageHTML: productPage, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows $389.00 as a was price: add it to the Offer's priceSpecification as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice and price 389.00, beside the sale price (Google's sale-price markup)"},
		}},
		{"a StrikethroughPrice carries it", audittest.Input{Draft: struckThrough("389"), PageHTML: productPage, PageURL: productURL}, nil},
		{"two was prices", audittest.Input{Draft: noDraft, PageHTML: twoOnSale, PageURL: productURL}, nil},
		{"a was price on another page of the shop", audittest.Input{Draft: noDraft, PageHTML: salePage, PageURL: salePageURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(productPage), PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.WasPrice{}, tc.in, tc.want) })
	}
}
