package figures_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/figures"
)

func TestDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a defined term", audittest.Input{Draft: figuresBare, PageHTML: figuresPage, PageURL: figuresURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page defines "Released value" but no DefinedTerm names it: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, description from the page's definition)`},
		}},
		{"a DefinedTerm", audittest.Input{Draft: withNodes(`{"@type":"DefinedTerm","name":"Released value"}`), PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"a noindex page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Noindex(figuresPage), PageURL: figuresURL}, nil},
		{"the page's text alone", audittest.Input{Draft: figuresBare, PageText: "Moving a 2-bedroom home. Released value, the basic cover a mover includes, pays. Your move takes 4–6.5 hours.", PageURL: figuresURL}, nil},
		{"no page", audittest.Input{Draft: figuresBare, PageURL: figuresURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, figures.Definitions{}, tc.in, tc.want) })
	}
}

// A cookie notice's "Rejecting these cookies means we won't know ..." is a
// consequence, not a term's definition (oliverbonas.com).
func TestDefinitionsSkipConsequences(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Returns</h1>
<p>Performance cookies. Rejecting these cookies means we won’t know that you have visited us before.</p>
<p>Social cookies. Rejecting these cookies means that you might not be able to use or see these sharing tools.</p>
<p>Targeting cookies. Rejecting these cookies means you may see less relevant adverts.</p></main></body></html>`
	audittest.Expect(t, figures.Definitions{}, audittest.Input{Draft: figuresBare, PageHTML: page, PageURL: figuresURL}, nil)
}
