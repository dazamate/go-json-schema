package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

func TestCitations(t *testing.T) {
	ask := func(n, list, rental string) string {
		return `the page credits ` + n + ` sources (` + list + `) that no citation names: list each as a citation of the WebPage, Article or Dataset (a CreativeWork with name, url when linked, and publisher the Organization behind it, a NewsMediaOrganization for a news publisher` + rental + `)`
	}
	// a task's last line ends with a "Source:" credit and no full stop; the
	// next task, and the note under the list, are other blocks
	checklist := `<!doctype html><html><body><main><h1>Moving checklist</h1>
<ul><li><label><input type="checkbox"> <span>Keep your moving receipts</span></label>
<p>The IRS says you may be able to deduct them.</p>
<small>Source: <cite>Internal Revenue Service</cite></small></li>
<li><label><input type="checkbox"> <span>Register to vote at your new address</span></label>
<p>Do it well before the next election.</p></li></ul>
<p>Source: Bureau of Labor Statistics</p>
<p>Your ticks never leave this device.</p>
<p>Source: <cite>U-Haul</cite> and <a href="https://www.penske.com/">Penske</a>; read 2026-10-28.</p>
</main></body></html>`
	checklistDraft := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/checklist/","name":"Moving checklist",
		"citation":[{"@type":"CreativeWork","name":"Internal Revenue Service"},{"@type":"CreativeWork","name":"U-Haul"}]}`
	// a "Sources:" label on a line of its own credits the next block, or each
	// item of the list after it
	labelled := `<!doctype html><html><body><main><h1>Truck sizes</h1><p>A 15 ft truck holds two rooms.</p>
<p><strong>Sources:</strong></p><ul><li>U-Haul</li><li>Penske</li></ul>
<dl><dt>Source:</dt><dd>Bureau of Labor Statistics</dd></dl></main></body></html>`
	trucks := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/trucks/","name":"Truck sizes"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"cites and source lines, with a rental company", audittest.Input{Draft: rulesBare, PageHTML: rulesPage, PageURL: rulesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: ask("3", "HireAHelper, moveBuddha, U-Haul", ", an AutoRental for a vehicle rental company")},
		}},
		{"a source line ends at its block", audittest.Input{Draft: checklistDraft, PageHTML: checklist, PageURL: "https://movers.test/checklist/"}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: ask("2", "Bureau of Labor Statistics, U-Haul and Penske", ", an AutoRental for a vehicle rental company")},
		}},
		{"a Sources: label credits the next block", audittest.Input{Draft: trucks, PageHTML: labelled, PageURL: "https://movers.test/trucks/"}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: ask("3", "U-Haul, Penske, Bureau of Labor Statistics", ", an AutoRental for a vehicle rental company")},
		}},
		{"the sources marked up", audittest.Input{Draft: rulesMarked, PageHTML: rulesPage, PageURL: rulesURL}, nil},
		{"a noindex page", audittest.Input{Draft: rulesBare, PageHTML: audittest.Noindex(rulesPage), PageURL: rulesURL}, nil},
		{"the page's text alone", audittest.Input{Draft: rulesBare, PageText: "What it costs. Source: HireAHelper.", PageURL: rulesURL}, nil},
		{"no page", audittest.Input{Draft: rulesBare, PageURL: rulesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Citations{}, tc.in, tc.want) })
	}
}
