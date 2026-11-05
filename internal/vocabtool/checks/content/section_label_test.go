package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestSectionLabel(t *testing.T) {
	const body = `<p class="label">Wedding traditions</p><h1>Tea ceremony: steps and order</h1>`
	article := graph(`{"@type":"Article","headline":"Tea ceremony"}`)
	run(t, content.SectionLabel{}, append([]tc{
		{"a label above the h1", at(article, body), warn("$.@graph[1]", `the page labels the article "Wedding traditions" above its h1: set articleSection "Wedding traditions"`)},
		{"articleSection set", at(graph(`{"@type":"Article","headline":"Tea ceremony","articleSection":"wedding traditions"}`), body), nil},
		{"no Article", at(webPage, body), nil},
		// theurbanlist.com: the eyebrow links to its section
		{"a label that links to its section", at(article, `<p class="EntryDetailArticle-eyebrow"> <a href="https://x.test/new-zealand/news"> News </a> </p><h1>Tea ceremony</h1>`), warn("$.@graph[1]", `the page labels the article "News" above its h1: set articleSection "News"`)},
		{"a link back", at(article, `<p class="label"><a href="/blog/">Back to blog</a></p><h1>Tea ceremony</h1>`), nil},
		{"a trail of links", at(article, `<p class="label"><a href="/">Home</a> <a href="/weddings/">Weddings</a></p><h1>Tea ceremony</h1>`), nil},
		{"a link within a sentence", at(article, `<p class="label">By <a href="/authors/jo/">Jo</a></p><h1>Tea ceremony</h1>`), nil},
		// vacuumwars.com: a pill that is a tagline
		{"a tagline", at(article, `<p class="vwx-pill">Independently tested. Never sponsored</p><h1>Tea ceremony</h1>`), nil},
		{"a label of six words", at(article, `<p>One two three four five six</p><h1>Tea ceremony</h1>`), nil},
		{"no h1", at(article, `<p class="label">Wedding traditions</p>`), nil},
		textOnly(article, "Wedding traditions Tea ceremony"),
	}, hidden(article, body)...))
}
