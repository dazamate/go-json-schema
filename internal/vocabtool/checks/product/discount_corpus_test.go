package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// The banners are the wording of cultfurniture.com's sale page and
// royalcanadiancollective.com's FAQ: a sitewide promo, no product's discount.
func TestDiscountPromoBanner(t *testing.T) {
	page := func(p string) string {
		return `<!doctype html><html><head></head><body><header>` + cartLink + `</header><main><h1>Sale</h1>` + p + `</main></body></html>`
	}
	for _, tc := range []struct {
		name string
		html string
		want []audit.Issue
	}{
		{"a code banner", page(`<p>Extra 25% off everything with <strong>BLACK25</strong> | Extra 30% off Cult Classics with BLACK30</p>`), nil},
		{"a first-order offer", page(`<p>GET 15% OFF + FREE SHIPPING ON YOUR FIRST ORDER!*</p>`), nil},
		// gardenbuildingsdirect.co.uk: the code sits in another span
		{"a code banner in parts", page(`<div style="text-align:center" class="carousel-item active"><span style="color:#FFFFFF">BLACK FRIDAY! UP TO </span><span style="color:#FFFF00"><strong>55% OFF</strong></span><span style="color:#FFFFFF"> WHEN YOU USE CODE </span><span style="color:#FFFF00"><strong>BF50</strong></span></div>`), nil},
		{"a product's discount", page(`<p>Save 15% on the Belrose Cushion</p>`), []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page shows a discount ("Save 15%") but no was price the markup can carry: show the was price, and mark it up as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice`},
		}},
		{"the Reliance MRP badge", page(`<p>MRP ₹1,89,999.00 (Inclusive of all taxes) 16% OFF</p>`), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, product.Discount{}, audittest.Input{Draft: noDraft, PageHTML: tc.html, PageURL: notesURL}, tc.want)
		})
	}
}

// cultfurniture.com: a menu link "Clearance Outlet - 65% off" is in the nav.
func TestDiscountNavLink(t *testing.T) {
	page := `<!doctype html><html><head></head><body><header>` + cartLink + `<nav><ul><li><a href="/collections/outlet">Clearance Outlet - 65% off</a></li></ul></nav></header><main><h1>Sale</h1><p>Furniture, lighting and decor.</p></main></body></html>`
	audittest.Expect(t, product.Discount{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, nil)
}

// cultfurniture.com's mobile drawer repeats the link outside any nav element.
func TestDiscountDrawerLink(t *testing.T) {
	page := `<!doctype html><html><head></head><body><header>` + cartLink + `</header><main><h1>Sale</h1><div class="drw-Navigation"><a href="/collections/outlet" class="drw-Navigation_Link">Clearance Outlet - 65% off</a></div></main></body></html>`
	audittest.Expect(t, product.Discount{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, nil)
}
