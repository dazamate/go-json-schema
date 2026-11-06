package figures_test

// The fixtures the figure checks share: a guide with a drawn chart, a
// summary, a truck table, a defined term, measured quantities and options,
// and the drafts that go with it.

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

// figuresBare is the guide's WebPage and nothing else.
const figuresBare = `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/2-bedroom/","name":"Moving a 2-bedroom home"}`

// withNodes is the guide's WebPage and the nodes, as one graph.
func withNodes(nodes string) string {
	return `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://movers.test/2-bedroom/","name":"Moving a 2-bedroom home"},` + nodes + `]}`
}
