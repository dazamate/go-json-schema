package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// discountPage is a product page that gives a discount and no was price.
const discountPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Opal ring</h1><p>Save 20% this week: $149.00.</p></main></body></html>`

const discountURL = "https://harbour.test/products/opal-ring"

func TestDiscount(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a discount and no was price", audittest.Input{Draft: noDraft, PageHTML: discountPage, PageURL: discountURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page shows a discount ("Save 20%") but no was price the markup can carry: show the was price, and mark it up as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice`},
		}},
		{"a StrikethroughPrice carries a was price", audittest.Input{Draft: struckThrough("186.25"), PageHTML: discountPage, PageURL: discountURL}, nil},
		{"the page shows the was price", audittest.Input{Draft: noDraft, PageHTML: `<!doctype html><html><head></head><body><header>` + cartLink + `</header><main><h1>Opal ring</h1><p>Save 20%: <del>$186.25</del> $149.00.</p></main></body></html>`, PageURL: discountURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(discountPage), PageURL: discountURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(discountPage), PageURL: discountURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: discountURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Discount{}, tc.in, tc.want) })
	}
}
