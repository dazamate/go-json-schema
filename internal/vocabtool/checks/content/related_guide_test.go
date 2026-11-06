package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestRelatedGuide(t *testing.T) {
	const body = `<section id="guide"><p>In this guide</p><ul><li><a href="/a/">A</a></li><li><a href="/b/">B</a></li><li><a href="/c/">C</a></li></ul></section>`
	listed := graph(`{"@type":"ItemList","name":"Guide","itemListElement":[
		{"@type":"ListItem","position":1,"name":"A","url":"https://x.test/a/"},
		{"@type":"ListItem","position":2,"name":"B","url":"https://x.test/b/"},
		{"@type":"ListItem","position":3,"name":"C","url":"https://x.test/c/"}]}`)
	run(t, content.RelatedGuide{}, append([]tc{
		{"a guide's pages no ItemList lists", at(webPage, body), warn("html:#guide", `the page's "In this guide" block lists 3 pages of the site: add an ItemList of them (a ListItem for each with position, url and name) as the mainEntity of the block's WebPageElement (a SiteNavigationElement for a nav) in the WebPage's hasPart; a hub page whose purpose is the list is a CollectionPage with the ItemList as its mainEntity`)},
		// gov.uk names its guide nav by its label, over a "Contents" heading
		{"a guide nav named by its label", at(webPage, `<nav id="guide" aria-label="Pages in this guide" class="gem-c-contents-list"><h2 class="gem-c-contents-list__title">Contents</h2><ol><li aria-current="true">What Universal Credit is</li><li><a href="/a/">Eligibility</a></li><li><a href="/b/">What you'll get</a></li><li><a href="/c/">How to claim</a></li></ol></nav>`), warn("html:#guide", `the page's "Pages in this guide" block lists 3 pages of the site: add an ItemList of them (a ListItem for each with position, url and name) as the mainEntity of the block's WebPageElement (a SiteNavigationElement for a nav) in the WebPage's hasPart; a hub page whose purpose is the list is a CollectionPage with the ItemList as its mainEntity`)},
		{"pages an ItemList lists", at(listed, body), nil},
		{"two pages", at(webPage, `<section id="guide"><p>In this guide</p><ul><li><a href="/a/">A</a></li><li><a href="/b/">B</a></li></ul></section>`), nil},
		// the block is the related-links check's
		{"a related block of three pages", at(webPage, `<section id="more"><p>Keep reading</p><ul><li><a href="/a/">A</a></li><li><a href="/b/">B</a></li><li><a href="/c/">C</a></li></ul></section>`), nil},
		textOnly(webPage, "In this guide A B C"),
	}, hidden(webPage, body)...))
}
