package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// kasaustralia.com.au's theme prints every stock state; the product is in stock.
func TestSoldOutStateList(t *testing.T) {
	states := `<div><span>Item is in stock</span><span>Only <span>11</span> left in stock - Selling Fast</span><span>Item is out of stock</span><span>Item is unavailable</span></div>`
	audittest.Expect(t, product.SoldOut{}, audittest.Input{Draft: noDraft, PageHTML: shopPageWith(states), PageURL: notesURL}, nil)
	audittest.Expect(t, product.SoldOut{}, audittest.Input{Draft: noDraft, PageHTML: shopPageWith(`<p>Charcoal: out of stock</p>`), PageURL: notesURL}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page shows "out of stock": set that product's Offer availability https://schema.org/SoldOut (or OutOfStock)`},
	})
}
