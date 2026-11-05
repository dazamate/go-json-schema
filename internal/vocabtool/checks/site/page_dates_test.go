package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestPageDates(t *testing.T) {
	ask := func(quote, prop, iso string) string {
		return `the page dates itself (` + quote + `) but no node has that date: set ` + prop + ` ` + iso + ` on the WebPage or Article (dateModified for an update, datePublished for publication, lastReviewed for a review or check)`
	}
	terms := `<!doctype html><html><body><main><h1>Terms</h1><p>Last updated: June 2021</p><p>These terms apply.</p></main></body></html>`
	many := `<!doctype html><html><body><main><h1>Dates</h1><p>Published 1 January 2020. Updated 2 February 2020. Reviewed 3 March 2020. Checked 4 April 2020.</p></main></body></html>`
	dated := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/what-movers-cost/","name":"What movers cost","lastReviewed":"2026-10-28"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a check date", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: movingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: ask(`"checked 28 October 2026"`, "lastReviewed", "2026-10-28")},
		}},
		{"an update date with no day", audittest.Input{Draft: bareDraft, PageHTML: terms, PageURL: "https://movers.test/terms/"}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: ask(`"Last updated: June 2021"`, "dateModified", "2021-06")},
		}},
		{"three dates at most", audittest.Input{Draft: bareDraft, PageHTML: many, PageURL: "https://movers.test/dates/"}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: ask(`"Published 1 January 2020"`, "datePublished", "2020-01-01")},
			{Path: "html:body", Severity: audit.Warning, Message: ask(`"Updated 2 February 2020"`, "dateModified", "2020-02-02")},
			{Path: "html:body", Severity: audit.Warning, Message: ask(`"Reviewed 3 March 2020"`, "lastReviewed", "2020-03-03")},
		}},
		{"the page's text alone", audittest.Input{Draft: bareDraft, PageText: "Last updated: June 2021. These terms apply.", PageURL: "https://movers.test/terms/"}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: ask(`"Last updated: June 2021"`, "dateModified", "2021-06")},
		}},
		{"the date in the markup", audittest.Input{Draft: dated, PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(movingGuide), PageURL: movingURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: movingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.PageDates{}, tc.in, tc.want) })
	}
}
