package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestProduct(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", "<p>Cast in silver.</p>")
	carried := `{"@context":"https://schema.org","@type":"Product","name":"Art Deco ring"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"no Product", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page sells "Art Deco ring" but no Product describes it: add a Product as the ItemPage's mainEntity (name, image, description, sku, brand a Brand, offers an Offer with url, price, priceCurrency, availability, itemCondition and seller the business)`},
		}},
		// autodiscount.co.nz's vehicle page: its h1 is empty, its title names the car
		{"a product named only in the title", audittest.Input{Draft: noDraft, PageHTML: productPageWith("", "2015 Toyota Aqua 4.5 Grade, X-Urban | Car Dealer Auckland NZ | Auto Discount", "<p>$14,450</p>"), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page sells "2015 Toyota Aqua 4.5 Grade, X-Urban" but no Product describes it: add a Product as the ItemPage's mainEntity (name, image, description, sku, brand a Brand, offers an Offer with url, price, priceCurrency, availability, itemCondition and seller the business)`},
		}},
		{"a Product is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Product{}, tc.in, tc.want) })
	}
}
