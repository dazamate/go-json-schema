package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// reliancedigital.in states its warranty after the word; royalcanadiancollective.com
// puts an adjective between the length and the word.
func TestWarrantyWording(t *testing.T) {
	for in, want := range map[string]string{
		`<p>Read Return Policy Manufacturer Warranty 1 Year Free Shipping</p>`:               "Warranty 1 Year",
		`<p>All Royal Canadian products carry a one-year limited warranty from the date</p>`: "one-year limited warranty",
		`<p>A 2 year manufacturer's warranty applies</p>`:                                    "2 year manufacturer's warranty",
	} {
		audittest.Expect(t, product.Warranty{}, audittest.Input{Draft: noDraft, PageURL: productURL, PageHTML: productPageWith("Phone", "Phone", in)}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page promises "` + want + `": set the Offer's warranty, a WarrantyPromise (durationOfWarranty a QuantitativeValue with unitCode ANN, warrantyScope)`},
		})
	}
	audittest.Expect(t, product.Warranty{}, audittest.Input{Draft: noDraft, PageURL: productURL,
		PageHTML: productPageWith("Phone", "Phone", `<p>Extended Warranty Plan for smartphones (6 Months)</p>`)}, nil)
}
