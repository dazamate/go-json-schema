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
		{"a sentence that ends in minutes before Read", at(article, `<p>Bake for 15 minutes. Read the notes below first.</p>`), nil},
		{"a time to read", at(article, `<p>Ann Lee · 5 min to read</p>`), warn("$.@graph[1]", `the page says "5 min to read" but no node has timeRequired: set timeRequired PT5M on the Article`)},
		{"a reading time label", at(article, `<p>Estimated reading time: 7 minutes</p>`), warn("$.@graph[1]", `the page says "reading time: 7 minutes" but no node has timeRequired: set timeRequired PT7M on the Article`)},
		{"an all-capitals read time", at(article, `<p>5 MIN. READ</p>`), warn("$.@graph[1]", `the page says "5 MIN. READ" but no node has timeRequired: set timeRequired PT5M on the Article`)},
		{"a title-case read time", at(article, `<p>7 Min Read</p>`), warn("$.@graph[1]", `the page says "7 Min Read" but no node has timeRequired: set timeRequired PT7M on the Article`)},
		{"a cook time is no reading time", at(article, `<p>Prep Time: 20 mins Cook Time: 10 mins</p>`), nil},
		{"no reading time on the page", at(article, "<p>Tea</p>"), nil},
	}, hidden(article, body)...))
}
