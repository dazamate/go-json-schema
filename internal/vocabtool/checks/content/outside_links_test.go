package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestOutsideLinks(t *testing.T) {
	const body = `<section id="friends"><h2>Friends</h2><a href="https://florist.test/">Florist</a> <a href="https://cake.test/">Cake</a> <a href="https://band.test/">Band</a></section>`
	named := graph(`{"@type":"ItemList","name":"Friends","itemListElement":[
		{"@type":"ListItem","position":1,"item":{"@type":"Florist","name":"Florist","url":"https://florist.test/"}},
		{"@type":"ListItem","position":2,"item":{"@type":"Bakery","name":"Cake","url":"https://cake.test/"}},
		{"@type":"ListItem","position":3,"item":{"@type":"MusicGroup","name":"Band","url":"https://band.test/"}}]}`)
	// no canonical link and no address: the page has no site to be outside of
	noBase := audittest.Input{Draft: webPage, PageHTML: `<!doctype html><html><head><title>Tea</title></head><body>` + body + `</body></html>`}
	run(t, content.OutsideLinks{}, append([]tc{
		{"three links to other sites", at(webPage, body), warn("html:#friends", "3 links to outside sites here (Florist, Cake, Band) name no node: when they are businesses or people the page features, add an ItemList of them to the WebPage's hasPart, each with url and sameAs; when they are sources, list them as citation")},
		{"links the draft names", at(named, body), nil},
		{"two links", at(webPage, `<section id="friends"><a href="https://florist.test/">Florist</a> <a href="https://cake.test/">Cake</a></section>`), nil},
		{"links to the site itself", at(webPage, `<section id="more"><a href="/a/">A</a> <a href="/b/">B</a> <a href="https://x.test/c/">C</a></section>`), nil},
		{"links to social profiles", at(webPage, `<section id="follow"><a href="https://www.facebook.com/ann">Facebook</a> <a href="https://www.instagram.com/ann">Instagram</a> <a href="https://www.tiktok.com/@ann">TikTok</a></section>`), nil},
		{"a social feed and share links", at(webPage, `<section id="feed"><a href="https://www.instagram.com/p/C9nT0D3xDBK/"><img src="/a.jpg" alt=""></a> <a href="https://www.instagram.com/reel/C91FjaURGZw/">Reel</a> <a href="https://twitter.com/home?status=https://ann.test/">Twitter</a> <a href="https://www.youtube.com/watch?v=abc">Video</a></section>`), nil},
		{"a page with no address", noBase, nil},
		textOnly(webPage, "Friends Florist Cake Band"),
	}, hidden(webPage, body)...))
}
