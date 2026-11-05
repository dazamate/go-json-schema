package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestUnitPrice(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"UnitPriceSpecification","price":4.5,"priceCurrency":"AUD","referenceQuantity":{"@type":"QuantitativeValue","value":100,"unitCode":"GRM"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a price per 100 g", audittest.Input{Draft: noDraft, PageHTML: productPage, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives a unit price ("$4.50 per 100 g"): add a UnitPriceSpecification with that price and referenceQuantity (a QuantitativeValue with value and unitCode) to the Offer's priceSpecification`},
		}},
		{"a referenceQuantity carries it", audittest.Input{Draft: carried, PageHTML: productPage, PageURL: productURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(productPage), PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.UnitPrice{}, tc.in, tc.want) })
	}
}
