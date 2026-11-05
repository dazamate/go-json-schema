package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// A collection called "Family favourites" (jamieoliver.com) or "Bestsellers &
// Favourites" (oliverbonas.com) saves nothing; "Add to Favourites"
// (autodiscount.co.nz) does.
func TestWantActionFavouritesCollection(t *testing.T) {
	for _, body := range []string{
		`<p><a href="/recipes/family-favourites/" aria-label="Family favourites">Family favourites</a></p>`,
		`<p><a href="/jewellery/jewellery-bestsellers">Bestsellers &amp; Favourites</a></p>`,
	} {
		audittest.Expect(t, product.WantAction{}, audittest.Input{Draft: noDraft, PageHTML: plainPageWith(body), PageURL: notesURL}, nil)
	}
	audittest.Expect(t, product.WantAction{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: plainPageWith(`<p><a href="#">Add to Favourites</a></p>`)}, []audit.Issue{
		{Path: "html:body > main > p > a", Severity: audit.Warning, Message: `the page lets a visitor save products to a wishlist ("Add to Favourites"): add a WantAction (target the wishlist page) to each Product's potentialAction`},
	})
}
