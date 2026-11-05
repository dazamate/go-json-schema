package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

const salePrice = `https://schema.org/StrikethroughPrice`

// kasaustralia.com.au: the theme calls the sale price "regular", and a similar
// product's struck price is another product's.
func TestWasPriceThemeRegularClass(t *testing.T) {
	page := productPageWith("Belrose Cushion", "Belrose Cushion", cartLink+buyForm+`
<div class="product__price"><span class="product__price--regular product__price--sale">$67.95 AUD</span><s class="product__price--compare">$79.95 AUD</s></div>
<section><h2>You may also like</h2><ul><li class="product-card"><a class="product-grid-item__price price" href="/products/linen-square-cushion-dijon"><span class="product-grid-item__price-new">$62.95 AUD</span> <s>$69.95 AUD</s></a></li><li class="product-card"><a href="/products/other">Other</a> $20.00</li></ul></section>`)
	audittest.Expect(t, product.WasPrice{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page shows $79.95 as a was price: add it to the Offer's priceSpecification as a UnitPriceSpecification with priceType ` + salePrice + ` and price 79.95, beside the sale price (Google's sale-price markup)`},
	})
}

// reliancedigital.in: an Indian shop's "MRP" in rupees is the was price.
func TestWasPriceMRP(t *testing.T) {
	page := productPageWith("Samsung Galaxy S26 Ultra", "Samsung Galaxy S26 Ultra", cartLink+buyForm+`
<div class="product-price">Offer Price ₹1,74,999.00</div><div class="product-marked-details"><span>MRP</span><span class="product-marked-price"> ₹1,89,999.00 </span><span>(Inclusive of all taxes)</span><span>16% OFF</span></div>`)
	audittest.Expect(t, product.WasPrice{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page shows ₹1,89,999.00 as a was price: add it to the Offer's priceSpecification as a UnitPriceSpecification with priceType ` + salePrice + ` and price 189999.00, beside the sale price (Google's sale-price markup)`},
	})
}
