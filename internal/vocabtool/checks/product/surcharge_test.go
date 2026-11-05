package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestSurcharge(t *testing.T) {
	charged := `{"@context":"https://schema.org","@type":"PaymentChargeSpecification","price":1.5,"priceCurrency":"AUD"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a card surcharge", audittest.Input{Draft: noDraft, PageHTML: productPage, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "incur a 1.5% surcharge": add a PaymentChargeSpecification (appliesToPaymentMethod the cards it applies to, and the charge) to the Offer's priceSpecification`},
		}},
		{"a PaymentChargeSpecification carries it", audittest.Input{Draft: charged, PageHTML: productPage, PageURL: productURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(productPage), PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Surcharge{}, tc.in, tc.want) })
	}
}
