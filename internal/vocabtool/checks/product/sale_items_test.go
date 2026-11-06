package product_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

var (
	// twoOnSale is the product page with a matching band on sale beside it.
	twoOnSale = strings.Replace(productPage, "GST included</p>", "GST included</p>\n<p>Matching band <del>$99.00</del> <ins>$79.00</ins></p>", 1)
	// salePage is another page of the shop that shows the ring set's was
	// price: it links the cart but sells no product of its own.
	salePage = strings.Replace(productPage, buyForm, "", 1)
)

const salePageURL = "https://harbour.test/pages/sale"

func TestSaleItems(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"two was prices", audittest.Input{Draft: noDraft, PageHTML: twoOnSale, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows was prices ($389.00, $99.00) on sale items: give each item's Offer a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice and its was price"},
		}},
		{"a was price on another page of the shop", audittest.Input{Draft: noDraft, PageHTML: salePage, PageURL: salePageURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows was prices ($389.00) on sale items: give each item's Offer a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice and its was price"},
		}},
		{"a StrikethroughPrice carries one", audittest.Input{Draft: struckThrough("389"), PageHTML: twoOnSale, PageURL: productURL}, nil},
		{"a product page's one was price", audittest.Input{Draft: noDraft, PageHTML: productPage, PageURL: productURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(twoOnSale), PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(twoOnSale), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.SaleItems{}, tc.in, tc.want) })
	}
}
