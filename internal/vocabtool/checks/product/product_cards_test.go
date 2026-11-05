package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// TestCardsAreFoundWhateverRanFirst: the material and size choice inside a
// related product's card are the card's, not this product's, so a check run
// alone stays quiet on them. The page's cards are found by whoever asks.
func TestCardsAreFoundWhateverRanFirst(t *testing.T) {
	page := productPageWith("Art deco ring", "Art deco ring", `<form action="/cart/add"><button>Add to cart</button></form>
<section id="also"><h2>You may also like</h2>`+
		`<div class="card"><a href="/products/gold/">Gold ring</a><span>$120.00</span><p>Material: Sterling silver</p><select name="size"><option>6</option><option>7</option></select></div>`+
		`<div class="card"><a href="/products/silver/">Silver ring</a><span>$90.00</span><p>Material: Sterling silver</p><select name="size"><option>6</option><option>7</option></select></div></section>`)
	in := audittest.Input{Draft: noDraft, PageHTML: page, PageURL: "https://harbour.test/products/ring/"}
	audittest.Expect(t, product.Details{}, in, nil)
	audittest.Expect(t, product.ProductGroup{}, in, nil)
}
