package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

// taxed is a draft whose PriceSpecification for the price shown sets
// valueAddedTaxIncluded.
func taxed(price, included string) string {
	return `{"@context":"https://schema.org","@type":"Offer","price":` + price + `,"priceCurrency":"AUD","priceSpecification":{"@type":"UnitPriceSpecification","price":` + price + `,"priceCurrency":"AUD","valueAddedTaxIncluded":` + included + `}}`
}

func TestTaxMismatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"tax included, the markup says not", audittest.Input{Draft: taxed("320", "false"), PageHTML: productPage, PageURL: productURL}, []audit.Issue{
			{Path: "$.priceSpecification.valueAddedTaxIncluded", Severity: audit.Warning, Message: `valueAddedTaxIncluded is false but the page says "GST included"`},
		}},
		{"the markup agrees", audittest.Input{Draft: taxed("320", "true"), PageHTML: productPage, PageURL: productURL}, nil},
		{"no price the draft sells is shown", audittest.Input{Draft: taxed("999", "false"), PageHTML: productPage, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: taxed("320", "false"), PageHTML: audittest.Noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: taxed("320", "false"), PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.TaxMismatch{}, tc.in, tc.want) })
	}
}
