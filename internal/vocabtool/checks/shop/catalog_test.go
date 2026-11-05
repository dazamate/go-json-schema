package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestCatalog(t *testing.T) {
	catalog := `{"@context":"https://schema.org","@type":"OnlineStore","hasOfferCatalog":{"@type":"OfferCatalog","name":"Rings"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a menu of categories", audittest.Input{Draft: noDraft, PageHTML: sitePage, PageURL: siteURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the site's menu lists 3 categories (Rings, Necklaces, Earrings): set the business's hasOfferCatalog, an OfferCatalog with a ListItem for each category naming its CollectionPage`},
		}},
		{"an OfferCatalog is there", audittest.Input{Draft: catalog, PageHTML: sitePage, PageURL: siteURL}, nil},
		{"two categories only", audittest.Input{Draft: noDraft, PageHTML: withTwoCategories(sitePage), PageURL: siteURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Opal ring. Cast by hand.`, PageURL: siteURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(sitePage), PageURL: siteURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: siteURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.Catalog{}, tc.in, tc.want) })
	}
}
