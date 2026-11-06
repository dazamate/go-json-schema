package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

// A business that never closes says so without a day or an hour.
func TestOpeningHoursAlwaysOpen(t *testing.T) {
	// jamesplumbingheating.com: the line is in the page's footer
	page := func(line string) string {
		return `<!doctype html><html><head></head><body><header><a href="/">James Plumbing &amp; Heating</a></header>
<main><h1>Transparent Pricing</h1><p>Call-out from £60.</p></main>
<footer><p>` + line + `</p></footer></body></html>`
	}
	for _, tc := range []struct{ line, said string }{
		{"Open 24 hours a day, 7 days a week", "Open 24 hours a day, 7 days a week"},
		{"We are open 7 days.", "open 7 days"},
		{"Open daily 8am to 6pm", "Open daily"},
	} {
		audittest.Expect(t, policies.OpeningHours{}, audittest.Input{Draft: noDraft, PageHTML: page(tc.line), PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives opening hours (` + `"` + tc.said + `"` + `): ` + hoursAdvice},
		})
	}
	// hive.com: "24/7" is a feature of a product
	audittest.Expect(t, policies.OpeningHours{}, audittest.Input{Draft: noDraft, PageHTML: page("AI 24/7 AI-powered. Open source."), PageURL: contactURL}, nil)
}
