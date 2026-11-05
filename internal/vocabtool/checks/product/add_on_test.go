package product_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestAddOn(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"Offer","price":320,"priceCurrency":"AUD","addOn":{"@type":"Offer","name":"Gift wrapping","price":5,"priceCurrency":"AUD"}}`
	// the gift wrapping without the cart link and the form that sells it
	unsold := strings.NewReplacer(cartLink, "", buyForm, "<p>Add gift wrapping for $5.00</p>").Replace(productPage)
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"paid gift wrapping", audittest.Input{Draft: noDraft, PageHTML: productPage, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page offers "Add gift wrapping for $5.00" as an extra: add it as the Offer's addOn, an Offer (name, price, priceCurrency)`},
		}},
		{"an addOn carries it", audittest.Input{Draft: carried, PageHTML: productPage, PageURL: productURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: unsold, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(productPage), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.AddOn{}, tc.in, tc.want) })
	}
}
