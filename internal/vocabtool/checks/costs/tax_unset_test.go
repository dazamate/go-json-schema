package costs_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

func TestTaxUnset(t *testing.T) {
	priced := `{"@context":"https://schema.org","@type":"Offer","price":320,"priceCurrency":"AUD"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"tax included, and no word of it in the markup", audittest.Input{Draft: priced, PageHTML: productPage, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "GST included" but no PriceSpecification sets valueAddedTaxIncluded: set it true`},
		}},
		{"plus tax", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "plus GST", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "plus GST" but no PriceSpecification sets valueAddedTaxIncluded: set it false`},
		}},
		{"All prices include GST", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "All prices include GST", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "include GST" but no PriceSpecification sets valueAddedTaxIncluded: set it true`},
		}},
		{"Prices exclude VAT", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "Prices exclude VAT", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "exclude VAT" but no PriceSpecification sets valueAddedTaxIncluded: set it false`},
		}},
		{"Prices +VAT", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "Prices +VAT", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "+VAT" but no PriceSpecification sets valueAddedTaxIncluded: set it false`},
		}},
		{"Prices + VAT", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "Prices + VAT", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "+ VAT" but no PriceSpecification sets valueAddedTaxIncluded: set it false`},
		}},
		{"Prices ex. GST", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "Prices ex. GST", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "ex. GST" but no PriceSpecification sets valueAddedTaxIncluded: set it false`},
		}},
		{"Prices excl. GST", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "Prices excl. GST", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "excl. GST" but no PriceSpecification sets valueAddedTaxIncluded: set it false`},
		}},
		{"Prices inc. GST", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "Prices inc. GST", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "inc. GST" but no PriceSpecification sets valueAddedTaxIncluded: set it true`},
		}},
		{"Price includes GST", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "Price includes GST", 1), PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's prices are "includes GST" but no PriceSpecification sets valueAddedTaxIncluded: set it true`},
		}},
		{"the markup sets it", audittest.Input{Draft: taxed("320", "true"), PageHTML: productPage, PageURL: productURL}, nil},
		{"no word of tax", audittest.Input{Draft: priced, PageHTML: strings.Replace(productPage, "GST included", "", 1), PageURL: productURL}, nil},
		{"no price the draft sells is shown", audittest.Input{Draft: noDraft, PageHTML: productPage, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: priced, PageHTML: audittest.Noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: priced, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.TaxUnset{}, tc.in, tc.want) })
	}
}
