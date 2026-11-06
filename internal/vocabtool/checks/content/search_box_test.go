package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestSearchBox(t *testing.T) {
	const body = `<form role="search" action="/search"><input type="search" name="q"></form>`
	withAction := graph(`{"@type":"WebSite","url":"https://x.test/","potentialAction":{"@type":"SearchAction","target":{"@type":"EntryPoint","urlTemplate":"https://x.test/search?q={search_term_string}"}}}`)
	run(t, content.SearchBox{}, append([]tc{
		{"a search box", at(webPage, body), warn("html:#main > form > input", "the page has a search box but no SearchAction: add one to the WebSite's potentialAction, with a target urlTemplate holding {search_term_string}")},
		{"a SearchAction in the draft", at(withAction, body), nil},
		{"a hidden search box", at(webPage, `<form hidden><input type="search" name="q"></form>`), nil},
		// the link is the other check's
		{"a link to the search page", at(webPage, `<a href="/search/" aria-label="Search">Search</a>`), nil},
		textOnly(webPage, "Search"),
	}, hidden(webPage, body)...))
}
