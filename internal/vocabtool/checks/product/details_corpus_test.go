package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// reliancedigital.in lists specifications as <li><span>label</span><span>value</span></li>;
// cultfurniture.com's cookie banner lists cookies as a dl of Name, Provider, Domain.
func TestDetailsSpecListAndCookies(t *testing.T) {
	cookies := `<div class="pd-cp-cookies"><ul><li class="pd-cp-cookies-list-item"><dl><dt>Name</dt><dd>_tracking_consent</dd><dt>Provider</dt><dd>Shopify</dd></dl></li></ul></div>`
	specs := `<ul><li class="specifications-list"><span>Customer care Phone</span><span><ul>1800-889-1055</ul></span></li><li class="specifications-list"><span>Country of origin</span><span>India</span></li><li class="specifications-list"><span>Rear Dual Camera</span><span>Yes</span></li></ul>`
	audittest.Expect(t, product.Details{}, audittest.Input{Draft: noDraft, PageURL: productURL, PageHTML: productPageWith("Phone", "Phone", cookies)}, nil)
	audittest.Expect(t, product.Details{}, audittest.Input{Draft: noDraft, PageURL: productURL, PageHTML: productPageWith("Phone", "Phone", specs)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page lists product details the markup lacks: Country of origin "India" (countryOfOrigin); set each on the Product`},
	})
}
