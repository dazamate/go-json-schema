package figures_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/figures"
)

func TestCharts(t *testing.T) {
	twoCharts := `<!doctype html><html><body><main><h1>Charts</h1>
<figure id="a"><figcaption>Chart one</figcaption><svg></svg></figure><figure id="b"><figcaption>Chart two</figcaption><svg></svg></figure></main></body></html>`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a drawn chart", audittest.Input{Draft: figuresBare, PageHTML: figuresPage, PageURL: figuresURL}, []audit.Issue{
			{Path: "html:#weather-temps", Severity: audit.Warning, Message: `the page draws the chart "Average high and low by month, °F" with no ImageObject: add one for each (caption word for word, encodingFormat image/svg+xml, about the Dataset it draws) as the image of the WebPage or the Dataset`},
		}},
		{"two drawn charts", audittest.Input{Draft: figuresBare, PageHTML: twoCharts, PageURL: figuresURL}, []audit.Issue{
			{Path: "html:#a", Severity: audit.Warning, Message: `the page draws 2 charts ("Chart one", "Chart two") with no ImageObject: add one for each (caption word for word, encodingFormat image/svg+xml, about the Dataset it draws) as the image of the WebPage or the Dataset`},
		}},
		{"an ImageObject with the caption", audittest.Input{Draft: withNodes(`{"@type":"ImageObject","caption":"Average high and low by month, °F","encodingFormat":"image/svg+xml"}`), PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"a noindex page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Noindex(figuresPage), PageURL: figuresURL}, nil},
		{"the page's text alone", audittest.Input{Draft: figuresBare, PageText: "Moving a 2-bedroom home. Released value, the basic cover a mover includes, pays. Your move takes 4–6.5 hours.", PageURL: figuresURL}, nil},
		{"no page", audittest.Input{Draft: figuresBare, PageURL: figuresURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, figures.Charts{}, tc.in, tc.want) })
	}
}
