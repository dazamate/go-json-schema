package site_test

// The fixtures the site checks share: a long guide, the drafts that go
// with it, and the address it is served at.

// movingGuide is a content site's guide: a header with a logo, a long
// article with sections, a dated source line and a toll-free number.
const movingGuide = `<!doctype html><html lang="en-US"><body>
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

const movingURL = "https://movers.test/what-movers-cost/"

// bareDraft is the guide's WebPage and nothing else.
const bareDraft = `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost"}`

// withNodes is the guide's WebPage and the nodes, as one graph.
func withNodes(nodes string) string {
	return `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost"},` + nodes + `]}`
}

// countdownPage has a moving checklist in phases, and a section of options
// with lists that is no checklist.
const countdownPage = `<!doctype html><html><body><main><h1>Moving</h1>
<section id="countdown"><h2>Your moving checklist</h2>
<div><h3><time datetime="P8W">8 weeks before</time></h3><ul><li><label><input type="checkbox"> <span>Get three estimates</span></label><p>Same details for each.</p></li><li><span>Book the movers</span></li></ul></div>
<div><h3><time datetime="P1D">The day before</time></h3><ul><li><span>Order boxes and tape</span></li></ul></div>
<div><h3>Moving day</h3><ul><li><span>Walk through every room</span></li></ul></div></section>
<section><h2>Ways to move</h2><h3>Movers</h3><ul><li>They lift</li></ul><h3>Truck</h3><ul><li>You drive</li></ul></section>
</main></body></html>`

const countdownURL = "https://movers.test/moving/"

// toolsPage has a quote tool, a tip calculator, a checklist of boxes and a
// form. Only the first two are tools.
const toolsPage = `<!doctype html><html><body><main><h1>Moving costs</h1>
<section id="calc"><header><p>Free moving estimate</p><p aria-live="polite"></p></header><h2>Where are you moving?</h2>
<label>From <input id="from"></label><label>To <input id="to"></label><p aria-live="polite"></p><button type="button">Continue</button></section>
<section id="tips"><h2>Tip calculator</h2><label>Crew <input type="range" id="crew"></label><output>3</output>
<label>Hours <input type="range" id="hours"></label><output>4</output><select id="rule"><option>10%</option></select><output>$40</output></section>
<section id="list"><h2>Checklist</h2><ul><li><label><input type="checkbox"> Pack</label></li><li><label><input type="checkbox"> Label</label></li><li><label><input type="checkbox"> Load</label></li></ul><p aria-live="polite">0 of 3 done</p></section>
<form method="post" action="/contact/"><input name="email"><button>Send</button></form>
</main></body></html>`

const toolsURL = "https://movers.test/costs/"

// toolsDraft is the tools page's WebPage and nothing else.
const toolsDraft = `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/costs/","name":"Moving costs"}`
