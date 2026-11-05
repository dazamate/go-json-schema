package shop_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestCatalogMenuMarkup(t *testing.T) {
	// taylorstitch.com: a menu link holds an icon whose title is "Nav chevron"
	chevron := `<svg class="icon icon-nav-chevron" focusable="false" aria-label="Nav chevron"><title>Nav chevron</title><use xlink:href="#icon-nav-chevron"></use></svg>`
	menu := strings.NewReplacer(">Rings</a>", ">Rings <span>"+chevron+"</span></a>", ">Necklaces</a>", ">Necklaces <span>"+chevron+"</span></a>").Replace(sitePage)
	audittest.Expect(t, shop.Catalog{}, audittest.Input{Draft: noDraft, PageHTML: menu, PageURL: siteURL}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the site's menu lists 3 categories (Rings, Necklaces, Earrings): set the business's hasOfferCatalog, an OfferCatalog with a ListItem for each category naming its CollectionPage`},
	})
	// sublimecandles.co.uk: WooCommerce's product-category pages
	woo := strings.NewReplacer("/collections/", "/product-category/").Replace(sitePage)
	audittest.Expect(t, shop.Catalog{}, audittest.Input{Draft: noDraft, PageHTML: woo, PageURL: siteURL}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the site's menu lists 3 categories (Rings, Necklaces, Earrings): set the business's hasOfferCatalog, an OfferCatalog with a ListItem for each category naming its CollectionPage`},
	})
}
