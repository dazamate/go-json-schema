package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// sublimecandles.co.uk's review form has a rating select; taylorstitch.com's
// size radios sit under the legend "Select Your Product Options" beside a
// variant-id select.
func TestProductGroupChoices(t *testing.T) {
	review := `<form action="/wp-comments-post.php" class="comment-form"><label for="rating">Your rating</label><select name="rating" id="rating"><option value="">Rate…</option><option value="5">Perfect</option><option value="4">Good</option></select><textarea name="comment"></textarea></form>`
	join := `<form action="/pages/trade"><select name="trade"><option value="">Business Sector*</option><option>Bars</option><option>Hotels</option></select><input type="email" name="email"></form>`
	radios := `<form action="/cart/add"><select name="id"><option value="1">XS - 36 | $118</option><option value="2">S - 38 | $118</option></select>
<fieldset><legend>Select Your Product Options</legend><input type="radio" id="r1" name="Size" value="XS - 36"><label for="r1">XS / 36</label><input type="radio" id="r2" name="Size" value="S - 38"><label for="r2">S / 38</label></fieldset></form>`
	audittest.Expect(t, product.ProductGroup{}, audittest.Input{Draft: noDraft, PageHTML: productPageWith("Candle", "Candle", review+join), PageURL: productURL}, nil)
	audittest.Expect(t, product.ProductGroup{}, audittest.Input{Draft: noDraft, PageHTML: productPageWith("Shirt", "Shirt", radios), PageURL: productURL}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page offers a choice of Size (XS / 36, S / 38): make the product a ProductGroup (productGroupID, variesBy https://schema.org/size) with a Product for each choice in hasVariant, each with its size and its own Offer`},
	})
}
