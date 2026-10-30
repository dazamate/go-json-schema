package vocabtool

import (
	"encoding/json"
	"testing"
)

// contentPage shows one of everything the content checks look for: a section
// label above the h1, a reading time, a content image, terms in another
// script, a counted heading, a quotation, prices with tax, a rating, a phone
// number and email, a registration number, outside links, a video, a search
// box, an enquiry form with a reply promise, a related-links block and a
// copyright line. It also holds what the checks must leave alone: a logo in
// the page header, a sentence in another script, a heading that counts
// minutes, and a search form.
const contentPage = `<!doctype html><html><head><title>Tea ceremony</title>
<link rel="canonical" href="https://x.test/tea/"></head><body>
<header id="site-header"><a href="/"><img src="/logo.png" alt="Ann Lee Weddings logo" width="40" height="40"></a></header>
<main id="main">
<nav aria-label="Breadcrumb"><ol><li><a href="/">Home</a></li></ol></nav>
<p class="label">Wedding traditions</p>
<h1>Tea ceremony: steps and order</h1>
<p class="byline">By Ann Lee · 7 min read</p>
<img src="/img/tea-640.webp" srcset="/img/tea-320.webp 320w, /img/tea-640.webp 640w" alt="A couple serving tea to elders" width="640" height="427">
<p>The Cantonese term is 敬茶 (ging3 caa4). Guests call out 飲勝 (jam2 sing3). The line 飲勝係廣東話嘅祝酒 (jam2 sing3 hai6 gwong2) is a sentence. Red dates (紅棗) sweeten the tea.</p>
<h2>The steps</h2>
<h3>Prepare the tea</h3><p>Boil water and steep the tea.</p>
<h3>Serve the elders</h3><p>Serve the eldest first.</p>
<h2>Four ways to say thanks</h2>
<ol><li>Bow</li><li>Smile</li><li>Toast</li><li>Hug</li></ol>
<h2>Plan 30 minutes</h2>
<figure><blockquote>“She made our day calm and fun.”</blockquote><figcaption><cite>@mei</cite></figcaption></figure>
<p>Most hosts charge $1,400 to $2,500 plus GST. Rated 4.8 out of 5 by couples.</p>
<p>Call <a href="tel:+61400111222">0400 111 222</a> or email <a href="mailto:ann@x.test">ann@x.test</a>.</p>
<p>ABN 12 345 678 901</p>
<section id="friends"><h2>Friends</h2>
<a href="https://florist.test/">Florist</a> <a href="https://cake.test/">Cake</a> <a href="https://band.test/">Band</a></section>
<iframe src="https://www.youtube.com/embed/abc123" title="Our tea ceremony"></iframe>
<form role="search" action="/search"><input type="search" name="q"></form>
<section id="enquire"><h2>Enquire</h2><p>We reply within 1 business day.</p><form method="POST"><input name="email"></form></section>
<section aria-labelledby="rel"><p class="label">Keep reading</p><h2 id="rel">More for your planning</h2><ul><li><a href="/door-games/">Door games</a></li></ul></section>
</main>
<footer id="site-footer"><p>© 2026 Ann Lee Weddings</p></footer>
</body></html>`

func checkPage(t *testing.T, draft, page string) Report {
	t.Helper()
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{PageHTML: page})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return r
}

func TestCheckFindsWhatThePageShowsThatTheMarkupLacks(t *testing.T) {
	r := checkPage(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"WebPage","@id":"https://x.test/tea/#webpage","url":"https://x.test/tea/","name":"Tea ceremony"},
		{"@type":"Article","headline":"Tea ceremony: steps and order","author":{"@type":"Person","name":"Ann Lee"}},
		{"@type":"HowTo","name":"Tea ceremony","step":[
			{"@type":"HowToStep","name":"Prepare the tea","text":"Boil water and steep the tea.","url":"https://x.test/tea/#step-1"},
			{"@type":"HowToStep","name":"Pour for the family"}]},
		{"@type":"Organization","@id":"https://x.test/#org","name":"Ann Lee Weddings","telephone":"+61 2 9999 0000","email":"hello@x.test",
			"potentialAction":{"@type":"AskAction","target":{"@type":"EntryPoint","urlTemplate":"https://x.test/tea/#enquire"}},
			"makesOffer":{"@type":"Offer","itemOffered":{"@type":"Service","name":"Hosting"}}},
		{"@type":"ImageObject","url":"https://x.test/img/tea-640.webp"},
		{"@type":"Intangible","name":"Misc"},
		{"@type":"Review","reviewBody":"Great host","itemReviewed":{"@id":"https://x.test/#org"}},
		{"@type":"WebSite","publisher":{"@id":"https://x.test/#org"}}]}`, contentPage)
	for _, w := range []struct{ path, msg string }{
		// the markup's text must be on the page
		{"$.@graph[2].step[1].name", `HowToStep name "Pour for the family" is not on the page`},
		{"$.@graph[6].reviewBody", `Review reviewBody "Great host" is not on the page`},
		// anchors must be ids on the page
		{"$.@graph[2].step[0].url", "#step-1, which is no id on the page"},
		// what the page shows must be in the markup
		{"$.@graph[0]", `"© 2026" but the markup has no copyrightYear 2026 and copyrightHolder`},
		{"$.@graph[1]", `"7 min read" but no node has timeRequired: set timeRequired PT7M`},
		{"$.@graph[1]", `labels the article "Wedding traditions" above its h1`},
		{"$.@graph[1].author", "set givenName Ann and familyName Lee"},
		{"html:#main > iframe", "no VideoObject has it as contentUrl or embedUrl"},
		{"html:#main > form > input", "a search box but no SearchAction"},
		{"html:#main > p:nth-of-type(7) > a:nth-of-type(1)", "links tel:+61400111222 but no node has that telephone"},
		{"html:#main > p:nth-of-type(7) > a:nth-of-type(2)", "links mailto:ann@x.test but no node has that email"},
		{"$.@graph[3].telephone", "telephone +61 2 9999 0000 is not on the page"},
		{"$.@graph[3].email", "email hello@x.test is not on the page"},
		{"$.@graph[3].potentialAction.target", "posts with POST: set httpMethod POST"},
		{"$.@graph[3].makesOffer", "this Offer names no seller"},
		{"html:body", "shows ABN 12 345 678 901 but no taxID, vatID or identifier carries it"},
		{"$.@graph[5]", "a bare Intangible says little"},
		{"$.@graph[6]", "this Review has no author"},
		{"html:body", "defines 3 terms (敬茶, 飲勝, 紅棗) that no DefinedTerm names"},
		{"$.@graph[4]", "set width 640, height 427, a description from its alt text"},
		{"html:#main > h2:nth-of-type(2)", "counts 4 items"},
		{"html:#friends", "3 links to outside sites here (Florist, Cake, Band) name no node"},
		{"html:#main > section:nth-of-type(3)", "https://x.test/door-games/"},
		{"html:body", `promises "reply within 1 business day" but no ServiceChannel has processingTime`},
		{"html:body", "shows prices ($1,400, $2,500) that no price property carries"},
		{"html:#main > figure > blockquote", `quotes "“She made our day calm and fun.”" (@mei) and no node carries it`},
		{"html:body", "shows ratings (4.8) that no Rating or AggregateRating carries"},
	} {
		if !issueAt(r, Warning, w.path, w.msg) {
			t.Errorf("no warning %q at %s", w.msg, w.path)
		}
	}
	// a review of the publisher is valid, so only an info
	if !issueAt(r, Info, "$.@graph[6]", "Google shows no review stars for self-serving reviews") {
		t.Errorf("no self-serving review info")
	}
	// the header logo, the sentence in another script and the heading that
	// counts minutes are left alone
	for _, s := range []string{"Ann Lee Weddings logo", "飲勝係", "Plan 30 minutes"} {
		if n := issueCount(r, s); n != 0 {
			t.Errorf("%d issues mention %q", n, s)
		}
	}
	if t.Failed() {
		t.Logf("issues: %+v", r.Issues)
	}
}

func TestCheckPassesAPageWhoseMarkupShowsWhatThePageShows(t *testing.T) {
	r := checkPage(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"WebPage","@id":"https://x.test/tea/#webpage","url":"https://x.test/tea/","name":"Tea ceremony",
			"copyrightYear":2026,"copyrightHolder":{"@id":"https://x.test/#org"},"relatedLink":["https://x.test/door-games/"],
			"breadcrumb":{"@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Home","item":"https://x.test/"}]},
			"image":{"@type":"ImageObject","contentUrl":"https://x.test/img/tea-640.webp","width":640,"height":427,"description":"A couple serving tea to elders"},
			"video":{"@type":"VideoObject","name":"Our tea ceremony","embedUrl":"https://www.youtube.com/embed/abc123"},
			"hasPart":[
				{"@type":"WPHeader","cssSelector":"#site-header"},
				{"@type":"WPFooter","cssSelector":"#site-footer"},
				{"@type":"ItemList","name":"Four ways to say thanks","numberOfItems":4},
				{"@type":"ItemList","name":"Friends","itemListElement":[
					{"@type":"ListItem","position":1,"item":{"@type":"Florist","name":"Florist","url":"https://florist.test/"}},
					{"@type":"ListItem","position":2,"item":{"@type":"Bakery","name":"Cake","url":"https://cake.test/"}},
					{"@type":"ListItem","position":3,"item":{"@type":"MusicGroup","name":"Band","url":"https://band.test/"}}]},
				{"@type":"DefinedTermSet","name":"Tea ceremony terms","hasDefinedTerm":[
					{"@type":"DefinedTerm","name":"敬茶","alternateName":"ging3 caa4"},
					{"@type":"DefinedTerm","name":"飲勝","alternateName":"jam2 sing3"},
					{"@type":"DefinedTerm","name":"紅棗","alternateName":"red dates"}]}]},
		{"@type":"Article","headline":"Tea ceremony: steps and order","timeRequired":"PT7M","articleSection":"Wedding traditions",
			"author":{"@type":"Person","name":"Ann Lee","givenName":"Ann","familyName":"Lee"}},
		{"@type":"HowTo","name":"Tea ceremony","step":[
			{"@type":"HowToStep","name":"Prepare the tea","text":"Boil water and steep the tea."},
			{"@type":"HowToStep","name":"Serve the elders","text":"Serve the eldest first."}]},
		{"@type":"Organization","@id":"https://x.test/#org","name":"Ann Lee Weddings","telephone":"0400 111 222","email":"ann@x.test","taxID":"12345678901",
			"potentialAction":{"@type":"AskAction","target":{"@type":"EntryPoint","urlTemplate":"https://x.test/tea/#enquire","httpMethod":"POST"}},
			"makesOffer":{"@type":"Offer","seller":{"@id":"https://x.test/#org"},"itemOffered":{"@type":"Service","name":"Hosting",
				"availableChannel":{"@type":"ServiceChannel","name":"Enquiry form","processingTime":"P1D"}}},
			"aggregateRating":{"@type":"AggregateRating","ratingValue":4.8,"bestRating":5,"ratingCount":12}},
		{"@type":"Service","name":"Tea ceremony hosting","offers":{"@type":"AggregateOffer","lowPrice":1400,"highPrice":2500,"priceCurrency":"AUD",
			"priceSpecification":{"@type":"PriceSpecification","minPrice":1400,"maxPrice":2500,"priceCurrency":"AUD","valueAddedTaxIncluded":false}}},
		{"@type":"Review","reviewBody":"She made our day calm and fun.","author":{"@type":"Person","name":"@mei"},"itemReviewed":{"@id":"https://x.test/#org"}},
		{"@type":"WebSite","publisher":{"@id":"https://x.test/#org"},
			"potentialAction":{"@type":"SearchAction","target":{"@type":"EntryPoint","urlTemplate":"https://x.test/search?q={search_term_string}"}}}]}`, contentPage)
	if r.Warnings != 0 || r.Errors != 0 {
		t.Errorf("%d warnings, %d errors, want none: %+v", r.Warnings, r.Errors, r.Issues)
	}
}

func TestCheckWarnsOnANumberWrittenAsText(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"Article","headline":"Tea","wordCount":"900"},
		{"@type":"ListItem","position":"1","name":"Home"}]}`)
	if !issueAt(r, Warning, "$.@graph[0].wordCount", `wordCount takes a number: write 900, not "900"`) {
		t.Errorf("no number warning for wordCount: %+v", r.Issues)
	}
	// position takes Text as well as Integer
	if n := issueCount(r, "takes a number"); n != 1 {
		t.Errorf("%d number warnings, want 1: %+v", n, r.Issues)
	}
}
