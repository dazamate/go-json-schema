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
		{"a holder before the year", at(webPage, `<p>© Love It Landscaping 2023. All Rights Reserved</p>`), warn("$.@graph[0]", `the page shows "© Love It Landscaping 2023" but the markup has no copyrightYear 2023 and copyrightHolder: set both on the WebPage or the WebSite`)},
		{"a copyright of a holder", at(webPage, `<p>© Copyright of Sublime Candles 2026</p>`), warn("$.@graph[0]", `the page shows "© Copyright of Sublime Candles 2026" but the markup has no copyrightYear 2026 and copyrightHolder: set both on the WebPage or the WebSite`)},
		{"an at sign for the mark", at(webPage, `<p>Copyright @ 2026 TutorFinder.com.au</p>`), warn("$.@graph[0]", `the page shows "Copyright @ 2026" but the markup has no copyrightYear 2026 and copyrightHolder: set both on the WebPage or the WebSite`)},
		{"the year before the mark", at(webPage, `<p>2024 © Acme Pty Ltd</p>`), warn("$.@graph[0]", `the page shows "2024 ©" but the markup has no copyrightYear 2024 and copyrightHolder: set both on the WebPage or the WebSite`)},
		{"a range of years, the first in the markup", at(`{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea ceremony","copyrightYear":2009,"copyrightHolder":{"@type":"Organization","name":"Taylor"}}]}`, `<p>©2009–2026 Taylor Stitch</p>`), nil},
		{"the word copyright in a heading, no year", at(webPage, `<li>Copyright Designs Circuit layouts Plant breeder's rights What to do if someone uses your work</li><p>Updated 2024</p>`), nil},
		{"year and holder set", at(whole, body), nil},
		{"no copyright line", at(webPage, "<p>Tea</p>"), nil},
	}, hidden(webPage, body)...))
}
