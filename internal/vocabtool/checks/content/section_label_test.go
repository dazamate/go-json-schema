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
		{"a label that links", at(article, `<p class="label"><a href="/weddings/">Wedding traditions</a></p><h1>Tea ceremony</h1>`), nil},
		{"a label of six words", at(article, `<p>One two three four five six</p><h1>Tea ceremony</h1>`), nil},
		{"no h1", at(article, `<p class="label">Wedding traditions</p>`), nil},
		textOnly(article, "Wedding traditions Tea ceremony"),
	}, hidden(article, body)...))
}
