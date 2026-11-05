package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestFromPrice(t *testing.T) {
	offered := `{"@context":"https://schema.org","@type":"CollectionPage","offers":{"@type":"AggregateOffer","lowPrice":99,"priceCurrency":"AUD"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a listing's starting price", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: listingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Prices start from $99.00": give the CollectionPage offers, an AggregateOffer (lowPrice 99.00, highPrice, priceCurrency, offerCount)`},
		}},
		{"an AggregateOffer carries it", audittest.Input{Draft: offered, PageHTML: listing, PageURL: listingURL}, nil},
		{"a page that is no listing", audittest.Input{Draft: noDraft, PageHTML: listing, PageURL: homeURL}, nil},
		{"one card is no listing", audittest.Input{Draft: noDraft, PageHTML: listingPage(opalCard), PageURL: listingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(listing), PageURL: listingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: listingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.FromPrice{}, tc.in, tc.want) })
	}
}
