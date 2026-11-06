package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestWebSitePartOf(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a WebPage with no isPartOf", audittest.Input{Draft: draftOf(siteLangNode, pageNode), PageHTML: sitePage, PageURL: siteURL}, []audit.Issue{
			{Path: "$.@graph[1]", Severity: audit.Warning, Message: `the WebPage has no isPartOf: set it to the WebSite ({"@id": "https://harbour.test/#website"})`},
		}},
		{"isPartOf is there", audittest.Input{Draft: draftOf(siteLangNode, pagePartNode), PageHTML: sitePage, PageURL: siteURL}, nil},
		{"no WebSite", audittest.Input{Draft: draftOf(pageNode), PageHTML: sitePage, PageURL: siteURL}, nil},
		{"no WebPage", audittest.Input{Draft: draftOf(siteLangNode), PageHTML: sitePage, PageURL: siteURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Opal ring. Cast by hand.`, PageURL: siteURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(sitePage), PageURL: siteURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: siteURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.WebSitePartOf{}, tc.in, tc.want) })
	}
}
