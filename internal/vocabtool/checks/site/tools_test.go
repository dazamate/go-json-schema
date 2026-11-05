package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestTools(t *testing.T) {
	both := `{"@type":"WebApplication","name":"Moving estimate","url":"https://movers.test/costs/#calc"},
		{"@type":"WebApplication","name":"Tip calculator","url":"https://movers.test/costs/#tips"}`
	sitewide := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://movers.test/costs/","name":"Moving costs"},
		{"@type":"WebApplication","@id":"https://movers.test/#moving-cost-calculator","name":"Movers Guide Moving Cost Calculator"}]}`
	words := `<!doctype html><html><body><main><h1>Moving costs</h1>
<section id="calculator"><p>Free moving estimate</p><label>From <input id="from"></label><label>To <input id="to"></label><p aria-live="polite"></p><button type="button">Continue</button></section>
<section id="tip-calculator"><h2>Tip calculator</h2><input type="range"><output>3</output><input type="range"><output>4</output><select><option>10%</option></select></section>
</main></body></html>`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"two tools no app names", audittest.Input{Draft: toolsDraft, PageHTML: toolsPage, PageURL: toolsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page has interactive tools ("Free moving estimate" #calc, "Tip calculator" #tips) but no WebApplication names them: add one for each (the SoftwareApplication that runs in a browser: name, url the page's URL + #id, applicationCategory, operatingSystem "Any", offers price 0 with seller, isAccessibleForFree), and make it the WebPage's mainEntity when the tool is the page's purpose`},
		}},
		{"a sitewide app names the first by its words", audittest.Input{Draft: sitewide, PageHTML: words, PageURL: toolsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page has interactive tools ("Tip calculator" #tip-calculator) but no WebApplication names them: add one for each (the SoftwareApplication that runs in a browser: name, url the page's URL + #id, applicationCategory, operatingSystem "Any", offers price 0 with seller, isAccessibleForFree), and make it the WebPage's mainEntity when the tool is the page's purpose`},
		}},
		{"an app for each by its #id", audittest.Input{Draft: withNodes(both), PageHTML: toolsPage, PageURL: toolsURL}, nil},
		{"a page with no tool", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a noindex page", audittest.Input{Draft: toolsDraft, PageHTML: audittest.Noindex(toolsPage), PageURL: toolsURL}, nil},
		{"the page's text alone", audittest.Input{Draft: toolsDraft, PageText: "Moving costs. Free moving estimate. Tip calculator.", PageURL: toolsURL}, nil},
		{"no page", audittest.Input{Draft: toolsDraft, PageURL: toolsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.Tools{}, tc.in, tc.want) })
	}
}
