package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestSearchLink(t *testing.T) {
	const body = `<a href="/search/" aria-label="Search">Search</a>`
	withAction := graph(`{"@type":"WebSite","url":"https://x.test/","potentialAction":{"@type":"SearchAction","target":{"@type":"EntryPoint","urlTemplate":"https://x.test/search?q={search_term_string}"}}}`)
	run(t, content.SearchLink{}, append([]tc{
		{"a link to the search page", at(webPage, body), warn(`html:a[aria-label="Search"]`, "the page links the site's search page (https://x.test/search) but no SearchAction: add one to the WebSite's potentialAction, target an EntryPoint whose urlTemplate is the search page's address with {search_term_string} as the query")},
		{"a SearchAction in the draft", at(withAction, body), nil},
		{"a link to another page", at(webPage, `<a href="/search-tips/">Search tips</a>`), nil},
		{"a link to another site's search", at(webPage, `<a href="https://other.test/search/">Search</a>`), nil},
		// the search box is the other check's, and it takes the page's one finding
		{"a search box as well", at(webPage, body+`<input type="search" name="q">`), nil},
		textOnly(webPage, "Search"),
	}, hidden(webPage, body)...))
}
