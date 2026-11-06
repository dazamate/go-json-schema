package figures_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/figures"
)

func TestQuantities(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"measured quantities", audittest.Input{Draft: figuresBare, PageHTML: figuresPage, PageURL: figuresURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page's tools and figures give quantities (4–6.5 hours, 4,500–6,500 lb): mark each up as a QuantitativeValue (value, or minValue and maxValue, unitCode such as HUR, DAY, MON, SMI, LBR or FTQ) on the thing it measures; a property whose range is a Duration, Distance or Mass takes it as that type's text (PT4H, "10 mi", "6500 lb")`},
		}},
		{"the numbers in the markup", audittest.Input{Draft: withNodes(`{"@type":"QuantitativeValue","minValue":4,"maxValue":6.5,"unitCode":"HUR"},{"@type":"QuantitativeValue","minValue":4500,"maxValue":6500,"unitCode":"LBR"}`), PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"a shop's page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Selling(figuresPage), PageURL: figuresURL}, nil},
		{"a noindex page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Noindex(figuresPage), PageURL: figuresURL}, nil},
		{"the page's text alone", audittest.Input{Draft: figuresBare, PageText: "Moving a 2-bedroom home. Released value, the basic cover a mover includes, pays. Your move takes 4–6.5 hours.", PageURL: figuresURL}, nil},
		{"no page", audittest.Input{Draft: figuresBare, PageURL: figuresURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, figures.Quantities{}, tc.in, tc.want) })
	}
}
