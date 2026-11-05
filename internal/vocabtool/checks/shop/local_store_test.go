package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestLocalStore(t *testing.T) {
	jewellers := []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page shows the street address "12 Harbour Street, Sydney NSW 2000" but no business node: add the business as a JewelryStore (the Store subtype for its trade, a LocalBusiness: name, address, geo, openingHoursSpecification, telephone)`},
	}
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a street address and no business node", audittest.Input{Draft: noDraft, PageHTML: storePage, PageURL: storeURL}, jewellers},
		{"on a page that sells", audittest.Input{Draft: noDraft, PageHTML: audittest.Selling(storePage), PageURL: storeURL}, jewellers},
		{"a business node", audittest.Input{Draft: business, PageHTML: storePage, PageURL: storeURL}, nil},
		{"a shop with no street address", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(storePage), PageURL: storeURL}, nil},
		// the text shows the address, so only the missing document keeps this
		// one quiet
		{"the page's text alone", audittest.Input{Draft: noDraft, PageText: storeText, PageURL: storeURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: storeURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.LocalStore{}, tc.in, tc.want) })
	}
}
