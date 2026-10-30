package vocabtool

import "testing"

// figuresPage is a guide with a drawn chart, a summary, a truck table, a
// defined term and measured quantities.
const figuresPage = `<!doctype html><html><body><main><h1>Moving a 2-bedroom home</h1>
<article><p>A long move is priced by weight and miles, and the mover's cover decides what you get back if something breaks on the way.</p>
<p>Released value, the basic cover a mover includes at no charge, pays at most 60 cents a pound. Released value is the default on a cheap quote. Ask whether released value applies before you sign.</p></article>
<section id="weather"><h2>Mild months</h2><figure id="weather-temps"><figcaption>Average high and low by month, °F</figcaption><div><svg aria-hidden="true"></svg></div></figure></section>
<section><h2>Key takeaways</h2><aside aria-label="Key takeaways"><ul><li>Book early.</li><li>Pack the kitchen last.</li></ul></aside></section>
<section id="truck-picker"><h2>The truck to rent</h2><figure><table><caption>Truck sizes for a 2-bedroom home move</caption>
<tr><th>Size</th><th>Space</th><th>Rated load</th></tr>
<tr><th>Cargo van</th><td>250 cu ft</td><td>3,500 lb</td></tr><tr><th>15 ft truck</th><td>760 cu ft</td><td>6,300 lb</td></tr></table></figure></section>
<section id="options"><h2>Three ways to move</h2>
<article><h3>Rent a truck</h3><h4>For it</h4><ul><li>Cheap</li></ul><h4>Against it</h4><ul><li>You lift</li></ul></article>
<article><h3>Hire movers</h3><h4>For it</h4><ul><li>They lift</li></ul><h4>Against it</h4><ul><li>Dearer</li></ul></article></section>
<section id="crew"><h2>Crew math</h2><p>Your move takes <output>4–6.5 hours</output> and weighs <data value="6500">4,500–6,500 lb</data>.</p></section>
</main></body></html>`

const figuresURL = "https://movers.test/2-bedroom/"

func TestCheckFindsWhatAGuidesFiguresShow(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/2-bedroom/","name":"Moving a 2-bedroom home"}`
	r := checkSite(t, bare, figuresPage, figuresURL)
	expectWarnings(t, r, [][]string{
		{`chart "Average high and low by month, °F"`, "no ImageObject", "image/svg+xml"},
		{`"Key takeaways" block sums the page up`, "abstract", "SpeakableSpecification", "Statement"},
		{`"Truck sizes for a 2-bedroom home move" lists vehicles (Cargo van, 15 ft truck)`, "add a Vehicle"},
		{`defines "Released value"`, "DefinedTerm"},
		{"4–6.5 hours", "4,500–6,500 lb", "as a QuantitativeValue", "Distance", "Duration", "Mass"},
		{`weighs options (Rent a truck, Hire movers)`, "Guide", "reviewAspect"},
	})
	for _, w := range []struct{ path, msg string }{{"html:#weather-temps", "no ImageObject"}, {"html:#truck-picker", "add a Vehicle"}} {
		if !issueAt(r, Warning, w.path, w.msg) {
			t.Errorf("no %q at %s", w.msg, w.path)
		}
	}
	marked := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://movers.test/2-bedroom/","name":"Moving a 2-bedroom home",
		"speakable":{"@type":"SpeakableSpecification","cssSelector":"aside[aria-label=\"Key takeaways\"]"},
		"image":{"@type":"ImageObject","caption":"Average high and low by month, °F","encodingFormat":"image/svg+xml"}},
		{"@type":"Vehicle","name":"Cargo van"},{"@type":"Vehicle","name":"15 ft truck"},
		{"@type":"DefinedTerm","name":"Released value"},
		{"@type":"Guide","name":"Three ways to move","reviewAspect":["For it","Against it"]},
		{"@type":"QuantitativeValue","minValue":4,"maxValue":6.5,"unitCode":"HUR"},{"@type":"QuantitativeValue","minValue":4500,"maxValue":6500,"unitCode":"LBR"}]}`
	r = checkSite(t, marked, figuresPage, figuresURL)
	for _, w := range []string{"no ImageObject", "sums the page up", "add a Vehicle", "DefinedTerm names", "defines \"Released value\"", "as a QuantitativeValue", "weighs options"} {
		if n := issueCount(r, w); n != 0 {
			t.Errorf("%q with the figures marked up: %+v", w, r.Issues)
		}
	}
}

func TestCheckLeavesAShopsFiguresAlone(t *testing.T) {
	for _, key := range []string{"qod-product-variants", "qod-category", "qod-home"} {
		r := checkShop(t, key, "")
		for _, w := range []string{"as a QuantitativeValue", "add a Vehicle", "sums the page up"} {
			if n := issueCount(r, w); n != 0 {
				t.Errorf("%s: %q: %+v", key, w, r.Issues)
			}
		}
	}
}
