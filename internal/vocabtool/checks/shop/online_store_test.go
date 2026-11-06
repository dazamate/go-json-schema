package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestOnlineStore(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a shop with no business node", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the site sells online but the page has no business node: add an OnlineStore (the OnlineBusiness subtype of Organization for a shop that trades online) as the WebSite's publisher, with name, url, logo, sameAs, email, taxID, hasMerchantReturnPolicy and hasShippingService"},
		}},
		{"a business node", audittest.Input{Draft: business, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		// a street address makes it a local store, even on a page that sells
		{"a shop with a street address", audittest.Input{Draft: noDraft, PageHTML: audittest.Selling(storePage), PageURL: storeURL}, nil},
		{"a site that sells nothing", audittest.Input{Draft: noDraft, PageHTML: guidePage, PageURL: guideURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(aboutPage), PageURL: aboutURL}, nil},
		// a page known only by its text cannot sell either
		{"the page's text alone", audittest.Input{Draft: noDraft, PageText: "About us. We make rings by hand.", PageURL: aboutURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: aboutURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.OnlineStore{}, tc.in, tc.want) })
	}
}
