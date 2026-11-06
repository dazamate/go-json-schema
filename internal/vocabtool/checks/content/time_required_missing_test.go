package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestTimeRequiredMissing(t *testing.T) {
	const body = `<p>By Ann Lee · 7 min read</p>`
	const msg = `the page says "7 min read" but no node has timeRequired: set timeRequired PT7M on the Article`
	article := graph(`{"@type":"Article","headline":"Tea ceremony"}`)
	run(t, content.TimeRequiredMissing{}, append([]tc{
		{"an Article without it", at(article, body), warn("$.@graph[1]", msg)},
		{"no Article, so the WebPage", at(webPage, body), warn("$.@graph[0]", msg)},
		{"the text alone", audittest.Input{Draft: article, PageText: "By Ann Lee · 7 min read", PageURL: pageURL}, warn("$.@graph[1]", msg)},
		{"the time given", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT7M"}`), body), nil},
		// the wrong time is the other check's
		{"another time given", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT5M"}`), body), nil},
		{"no reading time on the page", at(article, "<p>Tea</p>"), nil},
	}, hidden(article, body)...))
}
