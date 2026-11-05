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
		{"links in the footer", at(webPage, `<footer id="foot"><a href="https://a.test/privacy">Privacy Policy</a> <a href="https://b.test/terms">Terms of Service</a> <a href="https://c.test/">Site by Awedience</a></footer>`), nil},
		{"links in a footer div", at(webPage, `<div class="site-footer" id="ft"><a href="https://a.test/">Trustindex system</a> <a href="https://b.test/">Site by Awedience</a> <a href="https://c.test/">Subscribe</a></div>`), nil},
		{"links in a cookie banner", at(webPage, `<div id="CybotCookiebotDialog"><a href="https://a.test/privacy">Learn more about this provider</a> <a href="https://b.test/privacy">Learn more about this provider</a> <a href="https://c.test/privacy">Learn more about this provider</a></div>`), nil},
		{"links in a dialog", at(webPage, `<div id="modal" role="dialog"><a href="https://a.test/">One</a> <a href="https://b.test/">Two</a> <a href="https://c.test/">Three</a></div>`), nil},
		{"links to the business's own other sites", at(webPage, `<section id="more"><a href="https://shop.x.co.uk/">Shop</a> <a href="https://help.x.com/">Help</a> <a href="https://www.x.org/about">About</a></section>`), nil},
		{"links to the account, jobs and help", at(webPage, `<section id="acct"><a href="https://my.other.test/login">Log in</a> <a href="https://jobs.other.test/">Careers</a> <a href="https://help.other.test/">Help Centre</a> <a href="https://other.test/register">Register</a></section>`), nil},
		{"links to the app stores", at(webPage, `<div id="apps"><a href="https://apps.apple.com/us/app/x/id1">Download on the App Store</a> <a href="https://play.google.com/store/apps/details?id=x">Get it on Google Play</a> <a href="https://itunes.apple.com/x">iTunes</a></div>`), nil},
		{"citations in a body whose class names a menu", at(webPage, `<article id="post"><a href="https://rocket.test/learn/28-36-rule">28/36 rule</a> <a href="https://fha.test/loans">FHA loans</a> <a href="https://cfpb.test/pmi">Private Mortgage Insurance</a></article>`), warn("html:#post", "3 links to outside sites here (28/36 rule, FHA loans, Private Mortgage Insurance) name no node: when they are businesses or people the page features, add an ItemList of them to the WebPage's hasPart, each with url and sameAs; when they are sources, list them as citation")},
		{"a page with no address", noBase, nil},
		textOnly(webPage, "Friends Florist Cake Band"),
	}, hidden(webPage, body)...))
}
