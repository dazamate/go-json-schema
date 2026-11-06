package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestPrivatePage(t *testing.T) {
	const note = ": search engines leave its structured data out, so the shop checks skip it"
	info := func(msg string) []audit.Issue {
		return []audit.Issue{{Path: "html:body", Severity: audit.Info, Message: msg}}
	}
	bare := func(title string) string {
		return `<!doctype html><html><head><title>` + title + `</title></head><body><main><h1>` + title + `</h1></main></body></html>`
	}
	run(t, content.PrivatePage{}, []tc{
		{"a cart", audittest.Input{Draft: webPage, PageHTML: bare("Your bag"), PageURL: "https://x.test/cart/"}, info("the page is a shopping cart" + note)},
		{"a checkout, which would be a CheckoutPage", audittest.Input{Draft: webPage, PageHTML: bare("Checkout"), PageURL: "https://x.test/checkout/"},
			info("the page is a checkout" + note + "; if you mark it up at all, its WebPage is a CheckoutPage")},
		{"a noindex search page, which would be a SearchResultsPage", audittest.Input{Draft: webPage, PageHTML: audittest.Noindex(bare("Search results")), PageURL: "https://x.test/search/"},
			info("the page is noindex" + note + "; if you mark it up at all, its WebPage is a SearchResultsPage")},
		{"a noindex page of no other kind", audittest.Input{Draft: webPage, PageHTML: audittest.Noindex(page("<p>Tea</p>")), PageURL: pageURL}, info("the page is noindex" + note)},
		{"the text of a cart page", audittest.Input{Draft: webPage, PageText: "Your bag", PageURL: "https://x.test/cart/"}, info("the page is a shopping cart" + note)},
		{"an indexed page", at(webPage, "<p>Tea</p>"), nil},
		{"no page", audittest.Input{Draft: webPage, PageURL: pageURL}, nil},
	})
}
