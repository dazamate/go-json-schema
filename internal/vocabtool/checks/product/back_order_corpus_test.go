package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// sublimecandles.co.uk: WooCommerce says an item in stock can still be
// back-ordered; it is in stock.
func TestBackOrderInStock(t *testing.T) {
	audittest.Expect(t, product.BackOrder{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>£ 10.99 4 in stock (can be backordered)</p>`)}, nil)
	audittest.Expect(t, product.BackOrder{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Available on back order</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page says "back order": set the Offer's availability https://schema.org/BackOrder`},
	})
}
