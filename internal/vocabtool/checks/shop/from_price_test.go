package shop_test

import (
	"strings"
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
		// a garden-buildings shop's prices run past a thousand pounds
		{"a starting price with grouped thousands", audittest.Input{Draft: noDraft, PageHTML: strings.Replace(listing, "Prices start from $99.00", "Prices start from £1,137.00", 1), PageURL: listingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Prices start from £1,137.00": give the CollectionPage offers, an AggregateOffer (lowPrice 1137.00, highPrice, priceCurrency, offerCount)`},
		}},
		{"a starting price in rupees", audittest.Input{Draft: noDraft, PageHTML: strings.Replace(listing, "Prices start from $99.00", "Starting at ₹1,49,999", 1), PageURL: listingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Starting at ₹1,49,999": give the CollectionPage offers, an AggregateOffer (lowPrice 149999, highPrice, priceCurrency, offerCount)`},
		}},
		{"a starting price after its amount", audittest.Input{Draft: noDraft, PageHTML: strings.Replace(listing, "Prices start from $99.00", "Prices start from 1.299 €", 1), PageURL: listingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Prices start from 1.299 €": give the CollectionPage offers, an AggregateOffer (lowPrice 1299, highPrice, priceCurrency, offerCount)`},
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
