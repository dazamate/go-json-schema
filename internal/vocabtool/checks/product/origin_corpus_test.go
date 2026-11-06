package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// totallyirishgifts.com names a county, then the country; reliancedigital.in
// lists "Country of origin" in its specifications.
func TestOriginCountryAfterRegion(t *testing.T) {
	audittest.Expect(t, product.Origin{}, audittest.Input{Draft: noDraft, PageURL: productURL,
		PageHTML: productPageWith("Celtic Legends Chess Set", "Chess", `<p>Designed and made in County Dublin, Ireland. Handcrafted in County Dublin</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page says "made in County Dublin, Ireland": set the Product's countryOfOrigin, a Country (name Ireland)`},
	})
	audittest.Expect(t, product.Origin{}, audittest.Input{Draft: noDraft, PageURL: productURL,
		PageHTML: productPageWith("Samsung Galaxy", "Samsung", `<ul><li>Customer care Phone 1800-889-1055</li><li><span>Country of origin</span><span>India</span></li><li>Customer care email</li></ul>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page says "Country of origin India": set the Product's countryOfOrigin, a Country (name India)`},
	})
}
