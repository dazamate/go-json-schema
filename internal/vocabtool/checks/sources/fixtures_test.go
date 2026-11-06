package sources_test

// The fixtures the rule, permit, agency, document and citation checks share.

// rulesPage cites sources, rules and agencies the way a moving guide does.
const rulesPage = `<!doctype html><html><body><main><h1>Moving to Boston</h1>
<section id="rules"><h2>Your rights on an interstate move</h2>
<p>You may watch every weighing of your shipment. Source: 49 CFR 375.513 (eCFR, read 2026-10-28)</p>
<p>A binding estimate covers the services it lists. Source: 49 CFR 375.403 (eCFR, read 2026-10-28)</p>
<p>More on the rules from the <a href="https://www.fmcsa.dot.gov/protect-your-move">Federal Motor Carrier Safety Administration</a>.</p></section>
<section id="local-rules"><h2>Parking the truck</h2><p>A standard Moving Truck Permit reserves two spaces for one day. Apply 10 business days before the move on the <a href="https://www.boston.gov/moving-truck-permit">City's moving truck parking page</a>.</p></section>
<section id="costs"><h2>What it costs</h2><p>Rentals run from 10 ft vans to 26 ft trucks. <small>(<cite>HireAHelper</cite>)</small></p>
<p>Trucks average 10 to 12 miles per gallon. <small>(<cite>moveBuddha</cite>)</small></p>
<p>A 15 ft truck holds two rooms. <small>(<cite>U-Haul</cite>)</small></p>
<p>Read FMCSA's <a href="https://www.fmcsa.dot.gov/sites/fmcsa.dot.gov/files/ready-to-move.pdf">Ready to Move? brochure</a>.</p>
<blockquote><p>They were quick.</p><cite>Sam</cite></blockquote></section>
</main></body></html>`

const rulesURL = "https://movers.test/boston/"

// rulesBare is the Boston guide's WebPage and nothing else.
const rulesBare = `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/boston/","name":"Moving to Boston"}`

// rulesMarked is the Boston guide with everything it cites marked up.
const rulesMarked = `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/boston/","name":"Moving to Boston",
	"citation":[{"@type":"Legislation","legislationIdentifier":"49 CFR 375.513"},{"@type":"Legislation","legislationIdentifier":"49 CFR 375.403"},
		{"@type":"CreativeWork","name":"HireAHelper"},{"@type":"CreativeWork","name":"moveBuddha"},{"@type":"CreativeWork","name":"U-Haul"},
		{"@type":"DigitalDocument","name":"Ready to Move?","url":"https://www.fmcsa.dot.gov/sites/fmcsa.dot.gov/files/ready-to-move.pdf"}],
	"mentions":[{"@type":"GovernmentOrganization","name":"Federal Motor Carrier Safety Administration","url":"https://www.fmcsa.dot.gov/"},
		{"@type":"GovernmentPermit","name":"Moving Truck Permit","issuedBy":{"@type":"GovernmentOrganization","name":"City of Boston","url":"https://www.boston.gov/"}}]}`
