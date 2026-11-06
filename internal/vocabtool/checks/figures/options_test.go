package figures_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/figures"
)

func TestOptions(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"two options with pros and cons", audittest.Input{Draft: figuresBare, PageHTML: figuresPage, PageURL: figuresURL}, []audit.Issue{
			{Path: "html:#options", Severity: audit.Warning, Message: `the page weighs options (Rent a truck, Hire movers) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)`},
		}},
		{"a roundup with Pros & Cons headings", audittest.Input{Draft: figuresBare, PageHTML: roundup(
			`<h2>Ecovacs Deebot X8 Pro Omni</h2><div class="btn-wrapper"><h3>Pros &amp; Cons</h3></div><ul class="proscons"><li><ul class="pros"><li>Vacuums and mops effectively</li></ul></li></ul>` +
				`<h2>Roborock Qrevo Curv</h2><div class="btn-wrapper"><h3>Pros &amp; Cons</h3></div><ul class="proscons"><li><ul class="pros"><li>Strong suction</li></ul></li></ul>`), PageURL: figuresURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page weighs options (Ecovacs Deebot X8 Pro Omni, Roborock Qrevo Curv) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)`},
		}},
		{"Pros and Cons labels in spans", audittest.Input{Draft: figuresBare, PageHTML: roundup(
			`<h3>Citi Double Cash Card</h3><div><span>Pros</span><ul><li>2% cash back</li></ul><span>Cons</span><ul><li>No intro APR</li></ul></div>` +
				`<h3>Wells Fargo Active Cash Card</h3><div><span>Pros</span><ul><li>Welcome bonus</li></ul><span>Cons</span><ul><li>Foreign fee</li></ul></div>`), PageURL: figuresURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page weighs options (Citi Double Cash Card, Wells Fargo Active Cash Card) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)`},
		}},
		{"one option's pros and cons", audittest.Input{Draft: figuresBare, PageHTML: roundup(`<h2>Heat pumps</h2><h3>Pros &amp; Cons</h3><ul><li>Efficient</li></ul>`), PageURL: figuresURL}, nil},
		{"a Guide", audittest.Input{Draft: withNodes(`{"@type":"Guide","name":"Three ways to move"}`), PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"a noindex page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Noindex(figuresPage), PageURL: figuresURL}, nil},
		{"the page's text alone", audittest.Input{Draft: figuresBare, PageText: "Moving a 2-bedroom home. Released value, the basic cover a mover includes, pays. Your move takes 4–6.5 hours.", PageURL: figuresURL}, nil},
		{"no page", audittest.Input{Draft: figuresBare, PageURL: figuresURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, figures.Options{}, tc.in, tc.want) })
	}
}

// roundup is a page of the body in a main element.
func roundup(body string) string {
	return `<!doctype html><html><head><title>Best</title></head><body><main>` + body + `</main></body></html>`
}
