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
		{"a Guide", audittest.Input{Draft: withNodes(`{"@type":"Guide","name":"Three ways to move"}`), PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"a noindex page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Noindex(figuresPage), PageURL: figuresURL}, nil},
		{"the page's text alone", audittest.Input{Draft: figuresBare, PageText: "Moving a 2-bedroom home. Released value, the basic cover a mover includes, pays. Your move takes 4–6.5 hours.", PageURL: figuresURL}, nil},
		{"no page", audittest.Input{Draft: figuresBare, PageURL: figuresURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, figures.Options{}, tc.in, tc.want) })
	}
}
