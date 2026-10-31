package vocabtool

import "testing"

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

func TestCheckFindsTheSourcesAPageCites(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/boston/","name":"Moving to Boston"}`
	r := checkSite(t, bare, rulesPage, rulesURL)
	expectWarnings(t, r, [][]string{
		{`cites the rules "49 CFR 375.513", "49 CFR 375.403"`, "add a Legislation", "legislationIdentifier", "LegislationObject", "UnofficialLegalValue"},
		{`explains the permit "Moving Truck Permit" (from www.boston.gov)`, "add a GovernmentPermit", "issuedBy", "GovernmentService"},
		{`links documents ("Ready to Move? brochure")`, "DigitalDocument", "application/pdf"},
		{`agencies "Federal Motor Carrier Safety Administration" (www.fmcsa.dot.gov), "boston.gov" (www.boston.gov)`, "as a GovernmentOrganization"},
		{"credits 3 sources (HireAHelper, moveBuddha, U-Haul)", "that no citation names", "CreativeWork", "AutoRental"},
	})
	if warns(r, "credits", "Sam") {
		t.Errorf("a quote's author read as a source: %+v", r.Issues)
	}
	marked := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/boston/","name":"Moving to Boston",
		"citation":[{"@type":"Legislation","legislationIdentifier":"49 CFR 375.513"},{"@type":"Legislation","legislationIdentifier":"49 CFR 375.403"},
			{"@type":"CreativeWork","name":"HireAHelper"},{"@type":"CreativeWork","name":"moveBuddha"},{"@type":"CreativeWork","name":"U-Haul"},
			{"@type":"DigitalDocument","name":"Ready to Move?","url":"https://www.fmcsa.dot.gov/sites/fmcsa.dot.gov/files/ready-to-move.pdf"}],
		"mentions":[{"@type":"GovernmentOrganization","name":"Federal Motor Carrier Safety Administration","url":"https://www.fmcsa.dot.gov/"},
			{"@type":"GovernmentPermit","name":"Moving Truck Permit","issuedBy":{"@type":"GovernmentOrganization","name":"City of Boston","url":"https://www.boston.gov/"}}]}`
	r = checkSite(t, marked, rulesPage, rulesURL)
	for _, w := range []string{"add a Legislation", "add a GovernmentPermit", "as a GovernmentOrganization", "no citation names", "DigitalDocument"} {
		if n := issueCount(r, w); n != 0 {
			t.Errorf("%q with the sources marked up: %+v", w, r.Issues)
		}
	}
}

// checklistPage ends a task's last line with a "Source:" credit and no full stop;
// the next task, and the note under the list, are other blocks.
const checklistPage = `<!doctype html><html><body><main><h1>Moving checklist</h1>
<ul><li><label><input type="checkbox"> <span>Keep your moving receipts</span></label>
<p>The IRS says you may be able to deduct them.</p>
<small>Source: <cite>Internal Revenue Service</cite></small></li>
<li><label><input type="checkbox"> <span>Register to vote at your new address</span></label>
<p>Do it well before the next election.</p></li></ul>
<p>Source: Bureau of Labor Statistics</p>
<p>Your ticks never leave this device.</p>
<p>Source: <cite>U-Haul</cite> and <a href="https://www.penske.com/">Penske</a>; read 2026-10-28.</p>
</main></body></html>`

// A "Source:" line ends where its block does: the next block's words are not
// part of the source's name.
func TestCheckSourceLineEndsAtItsBlock(t *testing.T) {
	draft := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/checklist/","name":"Moving checklist",
		"citation":[{"@type":"CreativeWork","name":"Internal Revenue Service"},{"@type":"CreativeWork","name":"U-Haul"}]}`
	r := checkSite(t, draft, checklistPage, "https://movers.test/checklist/")
	expectWarnings(t, r, [][]string{
		{"credits 2 sources (Bureau of Labor Statistics, U-Haul and Penske)", "that no citation names"},
	})
	for _, w := range []string{"Register to vote", "Your ticks"} {
		if warns(r, "credits", w) {
			t.Errorf("the next block read as part of a source: %+v", r.Issues)
		}
	}
}

// A "Sources:" label on a line of its own credits the next block, or each item
// of the list after it.
func TestCheckSourceLabelCreditsTheNextBlock(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Truck sizes</h1><p>A 15 ft truck holds two rooms.</p>
<p><strong>Sources:</strong></p><ul><li>U-Haul</li><li>Penske</li></ul>
<dl><dt>Source:</dt><dd>Bureau of Labor Statistics</dd></dl></main></body></html>`
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/trucks/","name":"Truck sizes"}`
	r := checkSite(t, bare, page, "https://movers.test/trucks/")
	expectWarnings(t, r, [][]string{
		{"credits 3 sources (U-Haul, Penske, Bureau of Labor Statistics)", "that no citation names"},
	})
}
