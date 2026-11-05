package shop_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

const harbourSort = `<select id="sort" name="sort_by"><option value="manual">Featured</option><option value="price-ascending" selected>Price, low to high</option></select>`

// withSort is the listing with its sort control replaced.
func withSort(sel string) string { return strings.Replace(listing, harbourSort, sel, 1) }

// jewelryexchange.com: a WooCommerce sort select is named "orderby".
// jackweirandsons.com: a Shopify one says "Date, new to old".
func TestSortOrderOtherControls(t *testing.T) {
	for _, tc := range []struct{ sel, text, member, path string }{
		{`<select name="orderby" class="orderby" aria-label="Shop order"><option value="popularity">Sort by popularity</option><option value="price" selected="selected">Sort by price: low to high</option></select>`, "Sort by price: low to high", "ItemListOrderAscending", `html:select[aria-label="Shop order"]`},
		{`<select name="sort_by" id="SortByBar"><option value="manual">Featured</option><option value="created-descending" selected="selected">Date, new to old</option></select>`, "Date, new to old", "ItemListOrderDescending", "html:#SortByBar"},
		{`<select aria-label="Sort"><option selected>Price: Highest first</option></select>`, "Price: Highest first", "ItemListOrderDescending", `html:select[aria-label="Sort"]`},
		{`<select name="o"><option>Featured</option></select><select name="order"><option selected>Price: Low - High</option></select>`, "Price: Low - High", "ItemListOrderAscending", "html:body > main > select:nth-of-type(2)"},
	} {
		audittest.Expect(t, shop.SortOrder{}, audittest.Input{Draft: noDraft, PageHTML: withSort(tc.sel), PageURL: listingURL}, []audit.Issue{
			{Path: tc.path, Severity: audit.Warning, Message: `the page sorts its products "` + tc.text + `": set the ItemList's itemListOrder https://schema.org/` + tc.member + ` (an ItemListOrderType)`},
		})
	}
	// best selling is no order, and a "border" is no sort control
	audittest.Expect(t, shop.SortOrder{}, audittest.Input{Draft: noDraft, PageHTML: withSort(`<select name="sort_by"><option value="best-selling" selected="">Best selling</option></select>`), PageURL: listingURL}, nil)
	audittest.Expect(t, shop.SortOrder{}, audittest.Input{Draft: noDraft, PageHTML: withSort(`<select name="border"><option selected>Oldest first</option></select>`), PageURL: listingURL}, nil)
}
