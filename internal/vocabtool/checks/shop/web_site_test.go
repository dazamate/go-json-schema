package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestWebSite(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"no WebSite", audittest.Input{Draft: noDraft, PageHTML: sitePage, PageURL: siteURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the draft has no WebSite: add the site's WebSite to every page (@id https://harbour.test/#website, url https://harbour.test/, name, inLanguage en-AU, publisher the business) and set the WebPage's isPartOf to it`},
		}},
		{"a WebSite is there", audittest.Input{Draft: draftOf(siteNode), PageHTML: sitePage, PageURL: siteURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Opal ring. Cast by hand.`, PageURL: siteURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(sitePage), PageURL: siteURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: siteURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.WebSite{}, tc.in, tc.want) })
	}
}
