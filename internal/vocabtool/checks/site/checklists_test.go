package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestChecklists(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/moving/","name":"Moving"}`
	done := `{"@context":"https://schema.org","@graph":[` + bare + `,{"@type":"HowTo","name":"Your moving checklist","step":[{"@type":"HowToSection","name":"8 weeks before","itemListElement":[{"@type":"HowToStep","name":"Get three estimates"}]}]}]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a checklist in phases", audittest.Input{Draft: bare, PageHTML: countdownPage, PageURL: countdownURL}, []audit.Issue{
			{Path: "html:#countdown", Severity: audit.Warning, Message: `the page's checklist "Your moving checklist" has 3 phases and 4 tasks: add a HowTo (name, step: a HowToSection for each phase with a HowToStep for each task, name and text word for word, a HowToTip for a task's note, a HowToSupply or HowToTool where a task names one, and itemListOrder https://schema.org/ItemListOrderAscending) in the WebPage's hasPart`},
		}},
		{"a HowTo with the first task", audittest.Input{Draft: done, PageHTML: countdownPage, PageURL: countdownURL}, nil},
		{"a noindex page", audittest.Input{Draft: bare, PageHTML: audittest.Noindex(countdownPage), PageURL: countdownURL}, nil},
		{"the page's text alone", audittest.Input{Draft: bare, PageText: "Your moving checklist. 8 weeks before: get three estimates.", PageURL: countdownURL}, nil},
		{"no page", audittest.Input{Draft: bare, PageURL: countdownURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.Checklists{}, tc.in, tc.want) })
	}
}

// TestChecklistsOtherNiches pins pages from other niches. A recipe blog wraps
// every section in one article: its "How to Store" paragraph is no checklist
// just because a "Tips" list sits further down (allrecipes.com). A wedding
// page counts down in months, one heading per phase and one per task
// (firstlookweddings.co.uk), and a plain list under timed headings counts the
// same on any trade's page.
func TestChecklistsOtherNiches(t *testing.T) {
	const url = "https://cookies.test/recipe/"
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"` + url + `","name":"Cookies"}`
	recipe := `<!doctype html><html><body><main><article><h1>Chocolate Chip Cookies</h1>
	<h2>How to Store Chocolate Chip Cookies</h2>
	<p>Store the cooled chocolate chip cookies in an airtight container at room temperature for up to a week.</p>
	<h2>Can You Freeze Chocolate Chip Cookies?</h2>
	<p>Yes! You can freeze baked chocolate chip cookies and chocolate chip cookie dough.</p>
	<h3>Tips</h3>
	<ul><li>Use room-temperature butter.</li><li>Chill the dough.</li><li>Do not overbake.</li></ul>
	</article></main></body></html>`
	wedding := `<!doctype html><html><body><main><h1>Yes I Do: Ultimate 12 Month UK Wedding Planning Checklist</h1>
	<div><h2>12+ Months Before Wedding</h2><p>This foundation stage, big picture stuff.</p>
	<h3>Set a rough budget</h3><p>Decide what you can spend.</p>
	<h3>Draft your guest list</h3><p>Start with family.</p>
	<h3>Book your venue</h3><p>Venues book up fast.</p></div>
	<div><h2>9-11 Months Before Wedding</h2><p>Time for suppliers.</p>
	<h3>Book key suppliers</h3><p>Photographer first.</p>
	<h3>Go dress shopping</h3><p>Allow six months.</p></div>
	<h2>Contact us</h2></main></body></html>`
	homes := `<!doctype html><html><body><main><section id="steps"><h2>Home inspection checklist</h2>
	<h3>12 weeks before closing</h3><ul><li>Hire an inspector</li><li>Read the report</li></ul>
	<h3>2 weeks before closing</h3><ul><li>Check the repairs</li><li>Do the final walk-through</li></ul></section></main></body></html>`
	for _, tc := range []struct {
		name, html string
		want       []audit.Issue
	}{
		{"a recipe's how-to-store paragraph", recipe, nil},
		{"months-before headings with a heading per task", wedding, []audit.Issue{
			{Path: "html:body > main > div:nth-of-type(1) > h2", Severity: audit.Warning, Message: `the page's checklist "12+ Months Before Wedding" has 2 phases and 5 tasks: add a HowTo (name, step: a HowToSection for each phase with a HowToStep for each task, name and text word for word, a HowToTip for a task's note, a HowToSupply or HowToTool where a task names one, and itemListOrder https://schema.org/ItemListOrderAscending) in the WebPage's hasPart`},
		}},
		{"weeks before closing over lists", homes, []audit.Issue{
			{Path: "html:#steps", Severity: audit.Warning, Message: `the page's checklist "Home inspection checklist" has 2 phases and 4 tasks: add a HowTo (name, step: a HowToSection for each phase with a HowToStep for each task, name and text word for word, a HowToTip for a task's note, a HowToSupply or HowToTool where a task names one, and itemListOrder https://schema.org/ItemListOrderAscending) in the WebPage's hasPart`},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, site.Checklists{}, audittest.Input{Draft: bare, PageHTML: tc.html, PageURL: url}, tc.want)
		})
	}
}
