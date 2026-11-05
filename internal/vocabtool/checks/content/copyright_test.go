package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestCopyright(t *testing.T) {
	const body = `<p>© 2026 Ann Lee Weddings</p>`
	const msg = `the page shows "© 2026" but the markup has no copyrightYear 2026 and copyrightHolder: set both on the WebPage or the WebSite`
	whole := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea ceremony","copyrightYear":2026,"copyrightHolder":{"@type":"Organization","name":"Ann Lee Weddings"}}]}`
	yearOnly := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea ceremony","copyrightYear":2026}]}`
	other := `{"@context":"https://schema.org","@graph":[{"@type":"WebSite","url":"https://x.test/","name":"Ann"},{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea ceremony"}]}`
	run(t, content.Copyright{}, append([]tc{
		{"a year the markup lacks", at(webPage, body), warn("$.@graph[0]", msg)},
		{"the page's own WebPage, not the first", at(other, body), warn("$.@graph[1]", msg)},
		{"a year without a holder", at(yearOnly, body), warn("$.@graph[0]", msg)},
		{"the text alone", audittest.Input{Draft: webPage, PageText: "© 2026 Ann Lee Weddings", PageURL: pageURL}, warn("$.@graph[0]", msg)},
		{"year and holder set", at(whole, body), nil},
		{"no copyright line", at(webPage, "<p>Tea</p>"), nil},
	}, hidden(webPage, body)...))
}
