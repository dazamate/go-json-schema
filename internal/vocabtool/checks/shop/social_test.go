package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestSocial(t *testing.T) {
	sameAs := `{"@context":"https://schema.org","@type":"OnlineStore","sameAs":["https://www.instagram.com/harbourjewellers/"]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a profile no sameAs lists", audittest.Input{Draft: noDraft, PageHTML: sitePage, PageURL: siteURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the business's profiles (instagram.com/harbourjewellers) that no sameAs lists: add them to the business's sameAs, and a FollowAction for each (target the profile) to its potentialAction`},
		}},
		{"sameAs lists it", audittest.Input{Draft: sameAs, PageHTML: sitePage, PageURL: siteURL}, nil},
		{"a page with no profile link", audittest.Input{Draft: noDraft, PageHTML: withoutProfile(sitePage), PageURL: siteURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Opal ring. Cast by hand.`, PageURL: siteURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(sitePage), PageURL: siteURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: siteURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.Social{}, tc.in, tc.want) })
	}
}
