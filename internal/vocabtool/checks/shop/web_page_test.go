package shop_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

// shopPage is a page of a jewellery shop with a cart link in its header: the
// h1 and copy are its own.
func shopPage(title, h1, copy string) string {
	return `<!doctype html><html><head><title>` + title + ` · Harbour Jewellers</title></head><body>
<header><a href="/">Harbour Jewellers</a><nav><a href="/cart">Cart</a></nav></header>
<main><h1>` + h1 + `</h1><p>` + copy + `</p></main>
<footer><p>© 2026 Harbour Jewellers</p></footer></body></html>`
}

// The shop's about and contact pages, and where they are served.
var (
	aboutPage   = shopPage("About us", "About us", "We make rings by hand in our workshop.")
	contactPage = shopPage("Contact", "Contact us", "Write to us any time.")
)

const (
	aboutURL   = "https://harbour.test/about/"
	contactURL = "https://harbour.test/contact/"
)

func TestWebPage(t *testing.T) {
	webPage := `{"@context":"https://schema.org","@type":"WebPage","url":"https://harbour.test/contact/","name":"Contact us"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"an about page", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page has no WebPage: add a AboutPage for it (url, name, isPartOf the WebSite, mainEntity the business)"},
		}},
		{"a contact page", audittest.Input{Draft: noDraft, PageHTML: contactPage, PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page has no WebPage: add a ContactPage for it (url, name, isPartOf the WebSite, mainEntity the business)"},
		}},
		{"a page of no kind", audittest.Input{Draft: noDraft, PageHTML: strings.ReplaceAll(aboutPage, "About us", "Our rings"), PageURL: "https://harbour.test/rings-by-hand/"}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page has no WebPage: add a WebPage for it (url, name, isPartOf the WebSite, breadcrumb the BreadcrumbList when it has one)"},
		}},
		{"a WebPage in the draft", audittest.Input{Draft: webPage, PageHTML: contactPage, PageURL: contactURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(contactPage), PageURL: contactURL}, nil},
		// the kind comes from the address, so only the missing document keeps
		// this one quiet
		{"the page's text alone", audittest.Input{Draft: noDraft, PageText: "Contact us. Write to us any time.", PageURL: contactURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: contactURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.WebPage{}, tc.in, tc.want) })
	}
}
