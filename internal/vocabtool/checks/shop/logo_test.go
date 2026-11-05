package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestLogo(t *testing.T) {
	logo := `{"@context":"https://schema.org","@type":"OnlineStore","logo":"https://harbour.test/logo.svg"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a logo in the header", audittest.Input{Draft: noDraft, PageHTML: sitePage, PageURL: siteURL}, []audit.Issue{
			{Path: "html:body > header > a", Severity: audit.Warning, Message: `the site header shows the business's logo: set logo on the business, an ImageObject (url, width, height)`},
		}},
		{"a logo in the draft", audittest.Input{Draft: logo, PageHTML: sitePage, PageURL: siteURL}, nil},
		{"a header with no image", audittest.Input{Draft: noDraft, PageHTML: withoutLogo(sitePage), PageURL: siteURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Opal ring. Cast by hand.`, PageURL: siteURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(sitePage), PageURL: siteURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: siteURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.Logo{}, tc.in, tc.want) })
	}
}
