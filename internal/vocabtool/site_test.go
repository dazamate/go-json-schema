package vocabtool

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// checkSite checks a draft against a page served at an address.
func checkSite(t *testing.T, draft, page, address string) Report {
	t.Helper()
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{PageHTML: page, PageURL: address})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return r
}

// guidePage is a content site's guide: a header with a logo, a breadcrumb, a
// long article with sections, a dated source line and a toll-free number.
const guidePage = `<!doctype html><html lang="en-US"><body>
<header><a href="/"><svg aria-label="Movers Guide"></svg></a><nav aria-label="Main"><a href="/guides/">Guides</a></nav></header>
<nav aria-label="Breadcrumb"><ol><li><a href="/">Home</a></li><li>Moving costs</li></ol></nav>
<main><article><h1>What movers cost</h1>
<p>Movers charge by the hour for local moves and by weight for long ones. A crew of two with a truck is the usual booking for a small apartment, and most companies set a minimum of two or three hours. Fuel, stairs and long carries can add to the bill, so ask what each quote includes before you compare them side by side.</p>
<h2>Local moves</h2><p>Local moves are billed by the hour. The clock starts when the crew arrives and stops when the last box is in, and some companies add travel time between their depot and your home. Ask for the hourly rate, the minimum and the travel fee in writing.</p>
<h2>Long moves</h2><p>Long moves are priced by weight and distance. The mover weighs the truck before and after loading, and the difference sets the price along with the miles driven. A binding estimate fixes the price; a non-binding one can rise by up to ten percent at delivery.</p>
<p>Source: City Department of Transportation, checked <time datetime="2026-10-28">28 October 2026</time>.</p>
</article></main>
<footer><p>Call <a href="tel:+18334990939">(833) 499-0939</a></p><p>© 2026 Movers Guide</p></footer>
</body></html>`

const guideURL = "https://movers.test/what-movers-cost/"

func TestCheckAsksForThePageGraph(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost"}`
	r := checkSite(t, bare, guidePage, guideURL)
	expectWarnings(t, r, [][]string{
		{"the draft has no WebSite", "https://movers.test/#website", "isPartOf"},
		{"no node publishes the page", "Organization", "publisher"},
	})
	linked := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebSite","@id":"https://movers.test/#website","url":"https://movers.test/","name":"Movers Guide","inLanguage":"en-US","publisher":{"@id":"https://movers.test/#org"}},
		{"@type":"Organization","@id":"https://movers.test/#org","name":"Movers Guide","url":"https://movers.test/","address":{"@type":"PostalAddress","addressCountry":"US"}},
		{"@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost"},
		{"@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Home","item":"https://movers.test/"},{"@type":"ListItem","position":2,"name":"Moving costs"}]}]}`
	r = checkSite(t, linked, guidePage, guideURL)
	expectWarnings(t, r, [][]string{
		{"the WebPage has no isPartOf", "https://movers.test/#website"},
		{"the WebPage has no breadcrumb", "BreadcrumbList"},
	})
	unlinked := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebSite","url":"https://movers.test/","name":"Movers Guide"},
		{"@type":"Organization","name":"Movers Guide","url":"https://movers.test/"}]}`
	expectWarnings(t, checkSite(t, unlinked, guidePage, guideURL), [][]string{{"no node publishes the page", `set the Organization "Movers Guide" as the WebSite's publisher`}})
	for _, no := range []string{"has no WebSite", "no node publishes the page"} {
		if n := issueCount(r, no); n != 0 {
			t.Errorf("%q with the WebSite and its publisher in the draft: %+v", no, r.Issues)
		}
	}
}

func TestCheckTypesABusinessWithNoPremisesOnlineBusiness(t *testing.T) {
	draft := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebSite","@id":"https://movers.test/#website","url":"https://movers.test/","name":"Movers Guide","publisher":{"@id":"https://movers.test/#org"}},
		{"@type":"Organization","@id":"https://movers.test/#org","name":"Movers Guide","url":"https://movers.test/","telephone":"+18334990939"}]}`
	r := checkSite(t, draft, guidePage, guideURL)
	expectWarnings(t, r, [][]string{{`Organization "Movers Guide" shows no premises`, "type it OnlineBusiness"}})
	// a street address on the page makes it a local business instead
	shop := `<!doctype html><html><body><main><h1>Visit us</h1><p>Find us at 12 Harbour Street, Sydney NSW 2000.</p></main></body></html>`
	r = checkSite(t, draft, shop, "https://movers.test/contact/")
	if n := issueCount(r, "OnlineBusiness"); n != 0 {
		t.Errorf("a business with a street address told to be online only: %+v", r.Issues)
	}
	// nor with its address and hours in a footer outside main
	footer := `<!doctype html><html><body><main><h1>Hot water repairs</h1><p>We fix it the same day.</p></main>
<footer><p>12 Harbour Street, Sydney NSW 2000</p><p>Mon–Fri 8am–5pm</p></footer></body></html>`
	if r = checkSite(t, draft, footer, "https://movers.test/repairs/"); issueCount(r, "OnlineBusiness") != 0 {
		t.Errorf("a business with its address in the footer told to be online only: %+v", r.Issues)
	}
	// an Organization the page only mentions is not the business
	mention := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebSite","url":"https://movers.test/","name":"Movers Guide","publisher":{"@type":"Person","name":"Sam Lee"}},
		{"@type":"Organization","name":"FMCSA","url":"https://www.fmcsa.dot.gov/"}]}`
	r = checkSite(t, mention, guidePage, guideURL)
	if n := issueCount(r, "OnlineBusiness"); n != 0 {
		t.Errorf("a mentioned agency told to be an online business: %+v", r.Issues)
	}
}

func TestCheckFindsATollFreeNumber(t *testing.T) {
	draft := `{"@context":"https://schema.org","@type":"Organization","name":"Movers Guide","url":"https://movers.test/",
		"contactPoint":{"@type":"ContactPoint","telephone":"+18334990939","contactType":"customer support"}}`
	r := checkSite(t, draft, guidePage, guideURL)
	expectWarnings(t, r, [][]string{{"+18334990939 is toll-free", "contactOption TollFree"}})
	set := `{"@context":"https://schema.org","@type":"Organization","name":"Movers Guide","url":"https://movers.test/",
		"contactPoint":{"@type":"ContactPoint","telephone":"+18334990939","contactType":"customer support","contactOption":"https://schema.org/TollFree"}}`
	if r = checkSite(t, set, guidePage, guideURL); issueCount(r, "toll-free") != 0 {
		t.Errorf("contactOption TollFree is set: %+v", r.Issues)
	}
	local := `{"@context":"https://schema.org","@type":"Organization","name":"Movers Guide","contactPoint":{"@type":"ContactPoint","telephone":"+16195550100"}}`
	if r = checkSite(t, local, guidePage, guideURL); issueCount(r, "+16195550100 is toll-free") != 0 {
		t.Errorf("a local number called toll-free: %+v", r.Issues)
	}
}

func TestCheckFindsTheDatesAPageGivesItself(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost"}`
	r := checkSite(t, bare, guidePage, guideURL)
	expectWarnings(t, r, [][]string{{"checked 28 October 2026", "but no node has that date", "lastReviewed"}})
	dated := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost","lastReviewed":"2026-10-28"}`
	if r = checkSite(t, dated, guidePage, guideURL); issueCount(r, "no node has that date") != 0 {
		t.Errorf("the date is in the markup: %+v", r.Issues)
	}
	terms := `<!doctype html><html><body><main><h1>Terms</h1><p>Last updated: June 2021</p><p>These terms apply.</p></main></body></html>`
	r = checkSite(t, bare, terms, "https://movers.test/terms/")
	expectWarnings(t, r, [][]string{{"Last updated: June 2021", "no node has that date", "dateModified"}})
}

func TestCheckFindsAnArticleBody(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost"}`
	r := checkSite(t, bare, guidePage, guideURL)
	expectWarnings(t, r, [][]string{{"main copy is an article", "2 sections", "no node is an Article", "headline", "ReadAction"}})
	article := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost"},
		{"@type":"Article","headline":"What movers cost"}]}`
	if r = checkSite(t, article, guidePage, guideURL); issueCount(r, "no node is an Article") != 0 {
		t.Errorf("the draft has an Article: %+v", r.Issues)
	}
	terms := strings.Replace(guidePage, "<h1>What movers cost</h1>", "<h1>Terms and conditions</h1>", 1)
	if r = checkSite(t, bare, terms, "https://movers.test/terms-and-conditions/"); issueCount(r, "no node is an Article") != 0 {
		t.Errorf("the terms read as an article: %+v", r.Issues)
	}
	cards := `<!doctype html><html><body><main><h1>Compare</h1><article><h3>Movers</h3><p>A crew lifts and drives.</p></article><article><h3>Truck</h3><p>You drive.</p></article></main></body></html>`
	if r = checkSite(t, bare, cards, "https://movers.test/compare/"); issueCount(r, "no node is an Article") != 0 {
		t.Errorf("cards read as an article: %+v", r.Issues)
	}
}

func TestCheckFindsAGuidesChildPages(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Moving costs</h1><p>Start here.</p>
<section aria-label="In this guide"><nav aria-labelledby="g"><h2 id="g">In this guide</h2><ul>
<li><a href="/packing-cost/">Packing</a></li><li><a href="/hourly-rates/">Hourly rates</a></li><li><a href="/long-distance/">Long distance</a></li></ul></nav></section>
</main></body></html>`
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/moving-costs/","name":"Moving costs"}`
	r := checkSite(t, bare, page, "https://movers.test/moving-costs/")
	expectWarnings(t, r, [][]string{{`"In this guide" block lists 3 pages`, "ItemList", "hasPart", "CollectionPage"}})
	listed := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/moving-costs/","name":"Moving costs","hasPart":{"@type":"ItemList","itemListElement":[
		{"@type":"ListItem","position":1,"url":"https://movers.test/packing-cost/"},{"@type":"ListItem","position":2,"url":"https://movers.test/hourly-rates/"},{"@type":"ListItem","position":3,"url":"https://movers.test/long-distance/"}]}}`
	if r = checkSite(t, listed, page, "https://movers.test/moving-costs/"); issueCount(r, "In this guide") != 0 {
		t.Errorf("the guide's pages are listed: %+v", r.Issues)
	}
}

func TestCheckFindsAChecklist(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Moving</h1>
<section id="countdown"><h2>Your moving checklist</h2>
<div><h3><time datetime="P8W">8 weeks before</time></h3><ul><li><label><input type="checkbox"> <span>Get three estimates</span></label><p>Same details for each.</p></li><li><span>Book the movers</span></li></ul></div>
<div><h3><time datetime="P1D">The day before</time></h3><ul><li><span>Order boxes and tape</span></li></ul></div>
<div><h3>Moving day</h3><ul><li><span>Walk through every room</span></li></ul></div></section>
<section><h2>Ways to move</h2><h3>Movers</h3><ul><li>They lift</li></ul><h3>Truck</h3><ul><li>You drive</li></ul></section>
</main></body></html>`
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/moving/","name":"Moving"}`
	r := checkSite(t, bare, page, "https://movers.test/moving/")
	expectWarnings(t, r, [][]string{{`checklist "Your moving checklist" has 3 phases and 4 tasks`, "add a HowTo", "HowToSection", "HowToStep", "HowToTip", "HowToSupply", "HowToTool", "ItemListOrderAscending"}})
	if !slices.ContainsFunc(r.Issues, func(i Issue) bool { return i.Path == "html:#countdown" && strings.Contains(i.Message, "add a HowTo") }) {
		t.Errorf("the HowTo ask is not at the checklist: %+v", r.Issues)
	}
	if n := issueCount(r, "add a HowTo"); n != 1 {
		t.Errorf("%d HowTo asks, want the checklist's only: %+v", n, r.Issues)
	}
	done := `{"@context":"https://schema.org","@graph":[` + bare + `,{"@type":"HowTo","name":"Your moving checklist","step":[{"@type":"HowToSection","name":"8 weeks before","itemListElement":[{"@type":"HowToStep","name":"Get three estimates"}]}]}]}`
	if r = checkSite(t, done, page, "https://movers.test/moving/"); issueCount(r, "add a HowTo") != 0 {
		t.Errorf("the checklist has its HowTo: %+v", r.Issues)
	}
}

func TestCheckFindsAPagesTools(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Moving costs</h1>
<section id="calc"><header><p>Free moving estimate</p><p aria-live="polite"></p></header><h2>Where are you moving?</h2>
<label>From <input id="from"></label><label>To <input id="to"></label><p aria-live="polite"></p><button type="button">Continue</button></section>
<section id="tips"><h2>Tip calculator</h2><label>Crew <input type="range" id="crew"></label><output>3</output>
<label>Hours <input type="range" id="hours"></label><output>4</output><select id="rule"><option>10%</option></select><output>$40</output></section>
<section id="list"><h2>Checklist</h2><ul><li><label><input type="checkbox"> Pack</label></li><li><label><input type="checkbox"> Label</label></li><li><label><input type="checkbox"> Load</label></li></ul><p aria-live="polite">0 of 3 done</p></section>
<form method="post" action="/contact/"><input name="email"><button>Send</button></form>
</main></body></html>`
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/costs/","name":"Moving costs"}`
	r := checkSite(t, bare, page, "https://movers.test/costs/")
	expectWarnings(t, r, [][]string{{`interactive tools ("Free moving estimate" #calc, "Tip calculator" #tips)`, "but no WebApplication", "applicationCategory", "SoftwareApplication"}})
	if !slices.ContainsFunc(r.Issues, func(i Issue) bool { return i.Path == "html:#calc" && strings.Contains(i.Message, "has no QuoteAction") }) {
		t.Errorf("no QuoteAction ask at the quote tool: %+v", r.Issues)
	}
	if n := issueCount(r, "has no QuoteAction"); n != 1 {
		t.Errorf("%d QuoteAction asks, want the estimate's only: %+v", n, r.Issues)
	}
	done := `{"@context":"https://schema.org","@graph":[` + bare + `,
		{"@type":"WebApplication","name":"Moving estimate","url":"https://movers.test/costs/#calc","potentialAction":{"@type":"QuoteAction","target":{"@type":"EntryPoint","urlTemplate":"https://movers.test/costs/#calc"}}},
		{"@type":"WebApplication","name":"Tip calculator","url":"https://movers.test/costs/#tips"}]}`
	r = checkSite(t, done, page, "https://movers.test/costs/")
	for _, w := range []string{"no WebApplication", "has no QuoteAction"} {
		if n := issueCount(r, w); n != 0 {
			t.Errorf("%q with the tools marked up: %+v", w, r.Issues)
		}
	}
}

func TestCheckCountsAToolNamedByASitewideApp(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Moving costs</h1>
<section id="calculator"><p>Free moving estimate</p><label>From <input id="from"></label><label>To <input id="to"></label><p aria-live="polite"></p><button type="button">Continue</button></section>
<section id="tip-calculator"><h2>Tip calculator</h2><input type="range"><output>3</output><input type="range"><output>4</output><select><option>10%</option></select></section>
</main></body></html>`
	draft := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://movers.test/costs/","name":"Moving costs"},
		{"@type":"WebApplication","@id":"https://movers.test/#moving-cost-calculator","name":"Movers Guide Moving Cost Calculator"}]}`
	r := checkSite(t, draft, page, "https://movers.test/costs/")
	expectWarnings(t, r, [][]string{{`interactive tools ("Tip calculator" #tip-calculator)`, "no WebApplication"}})
}
