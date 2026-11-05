package shop_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestProductCount(t *testing.T) {
	counted := `{"@context":"https://schema.org","@type":"ItemList","numberOfItems":48}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a listing's product count", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: listingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "48 products": set numberOfItems 48 on the ItemList of its products`},
		}},
		{"an ItemList has the count", audittest.Input{Draft: counted, PageHTML: listing, PageURL: listingURL}, nil},
		{"a count below the cards shown", audittest.Input{Draft: noDraft, PageHTML: strings.Replace(listing, "48 products", "2 products", 1), PageURL: listingURL}, nil},
		{"a page that is no listing", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: homeURL}, nil},
		{"one card is no listing", audittest.Input{Draft: noDraft, PageHTML: listingPage(opalCard), PageURL: listingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(listing), PageURL: listingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: listingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.ProductCount{}, tc.in, tc.want) })
	}
}
