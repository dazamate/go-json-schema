package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// cultfurniture.com/collections/sale has a cart drawer's add-to-cart form and
// a "Kitchen Stools (60-70cm)" menu link; it is a collection, no product.
func TestProductCheckSkipsCollection(t *testing.T) {
	page := productPageWith("Sale", "Furniture Sale", `<nav><a href="/collections/kitchen-stools">Kitchen Stools (60-70cm)</a> <a href="/collections/stools">Bar Stools (65-75cm)</a></nav><form action="/cart" method="POST"><button>Add to cart</button></form>`)
	for _, u := range []string{"https://harbour.test/collections/sale"} {
		audittest.Expect(t, product.Product{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: u}, nil)
		audittest.Expect(t, product.Sizes{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: u}, nil)
	}
	audittest.Expect(t, product.Product{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: "https://harbour.test/collections/stools/products/oak-stool"}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page sells "Sale" but no Product describes it: add a Product as the ItemPage's mainEntity (name, image, description, sku, brand a Brand, offers an Offer with url, price, priceCurrency, availability, itemCondition and seller the business)`},
	})
}

// The same range on a product page itself is no size either.
func TestSizesRangeIsNoSize(t *testing.T) {
	page := productPageWith("Oak stool", "Oak stool", `<form action="/cart/add"><button>Add to cart</button></form><p>Kitchen Stools (60-70cm) and Bar Stools (65-75cm)</p>`)
	audittest.Expect(t, product.Sizes{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, nil)
}
