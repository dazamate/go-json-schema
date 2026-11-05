package figures_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/figures"
)

func TestSummary(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a Key takeaways block", audittest.Input{Draft: figuresBare, PageHTML: figuresPage, PageURL: figuresURL}, []audit.Issue{
			{Path: "html:body > main > section:nth-of-type(2)", Severity: audit.Warning, Message: `the page's "Key takeaways" block sums the page up: set its text as the Article's abstract, or point a SpeakableSpecification's cssSelector at it (the WebPage's speakable), or give each line as a Statement (text) in the WebPage's hasPart`},
		}},
		{"a speakable part", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/2-bedroom/","name":"Moving a 2-bedroom home","speakable":{"@type":"SpeakableSpecification","cssSelector":"aside"}}`, PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"an abstract", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Article","headline":"x","abstract":"Book early."}`, PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"a shop's page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Selling(figuresPage), PageURL: figuresURL}, nil},
		{"a noindex page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Noindex(figuresPage), PageURL: figuresURL}, nil},
		{"the page's text alone", audittest.Input{Draft: figuresBare, PageText: "Moving a 2-bedroom home. Released value, the basic cover a mover includes, pays. Your move takes 4–6.5 hours.", PageURL: figuresURL}, nil},
		{"no page", audittest.Input{Draft: figuresBare, PageURL: figuresURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, figures.Summary{}, tc.in, tc.want) })
	}
}
