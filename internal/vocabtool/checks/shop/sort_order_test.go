package shop_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestSortOrder(t *testing.T) {
	ordered := `{"@context":"https://schema.org","@type":"ItemList","itemListOrder":"https://schema.org/ItemListOrderAscending"}`
	sorted := []audit.Issue{
		{Path: "html:#sort", Severity: audit.Warning, Message: `the page sorts its products "Price, low to high": set the ItemList's itemListOrder https://schema.org/ItemListOrderAscending (an ItemListOrderType)`},
	}
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a listing sorted low to high", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: listingURL}, sorted},
		{"a sorted row on the home page", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: homeURL}, sorted},
		{"an ItemList has the order", audittest.Input{Draft: ordered, PageHTML: listing, PageURL: listingURL}, nil},
		{"no order chosen", audittest.Input{Draft: noDraft, PageHTML: strings.Replace(listing, " selected", "", 1), PageURL: listingURL}, nil},
		{"one card is no listing", audittest.Input{Draft: noDraft, PageHTML: listingPage(opalCard), PageURL: listingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(listing), PageURL: listingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: listingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.SortOrder{}, tc.in, tc.want) })
	}
}
