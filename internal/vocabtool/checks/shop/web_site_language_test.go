package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestWebSiteLanguage(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a WebSite with no language", audittest.Input{Draft: draftOf(siteNode), PageHTML: sitePage, PageURL: siteURL}, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: `this WebSite has no inLanguage: set it en-AU, the page's language`},
		}},
		{"inLanguage is there", audittest.Input{Draft: draftOf(siteLangNode), PageHTML: sitePage, PageURL: siteURL}, nil},
		{"no WebSite", audittest.Input{Draft: noDraft, PageHTML: sitePage, PageURL: siteURL}, nil},
		{"a page that names no language", audittest.Input{Draft: draftOf(siteNode), PageHTML: withoutLang(sitePage), PageURL: siteURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Opal ring. Cast by hand.`, PageURL: siteURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(sitePage), PageURL: siteURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: siteURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.WebSiteLanguage{}, tc.in, tc.want) })
	}
}
