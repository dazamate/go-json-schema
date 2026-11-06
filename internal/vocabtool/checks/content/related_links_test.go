package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestRelatedLinks(t *testing.T) {
	const body = `<section aria-labelledby="rel"><p class="label">Keep reading</p><h2 id="rel">More for your planning</h2><ul><li><a href="/door-games/">Door games</a></li></ul></section>`
	listed := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea ceremony","relatedLink":["https://x.test/door-games/"]}]}`
	run(t, content.RelatedLinks{}, append([]tc{
		{"a related block the WebPage does not list", at(webPage, body), warn("html:#main > section", `the page's "Keep reading" links are not in the WebPage's relatedLink: add https://x.test/door-games/`)},
		{"links the WebPage lists", at(listed, body), nil},
		{"a link to another site", at(webPage, `<section><p>Keep reading</p><ul><li><a href="https://other.test/x/">Elsewhere</a></li></ul></section>`), nil},
		{"a link to a place on the page", at(webPage, `<section><p>Keep reading</p><ul><li><a href="#top">Top</a></li></ul></section>`), nil},
		{"a card's read more call to action", at(webPage, `<section><p>Read more about benefits</p><a href="/c/cancellation-and-return-policy">Returns</a></section>`), nil},
		{"an excerpt's read more", at(webPage, `<article><h3>Mustard</h3><p>A short story about mustard.</p><span>Read more</span><a href="/blog/mustard/">here</a></article>`), nil},
		textOnly(webPage, "Keep reading Door games"),
	}, hidden(webPage, body)...))
}
