package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

func TestDataset(t *testing.T) {
	described := `{"@context":"https://schema.org","@graph":[
		{"@type":"Dataset","name":"Moving companies per 100,000 residents, 2023"},
		{"@type":"Dataset","name":"Hourly pay for hand laborers and material movers, May 2025"},
		{"@type":"Dataset","name":"Road miles to ten cities"},
		{"@type":"Dataset","name":"Homes by bedrooms, 2020-2024"}]}`
	ask := "but no Dataset describes it: add one (name and description from the caption and its note, creator or publisher the source, isBasedOn its URL, temporalCoverage from the years, spatialCoverage from the places, variableMeasured from the column headers)"
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"modules with a statistical source", audittest.Input{Draft: barePage, PageHTML: dataPage, PageURL: dataURL}, []audit.Issue{
			{Path: "html:#movers-nearby", Severity: audit.Warning, Message: `the page shows data from U.S. Census Bureau, County Business Patterns, 2023 ("Moving companies per 100,000 residents, 2023") ` + ask},
			{Path: "html:#mover-pay", Severity: audit.Warning, Message: `the page shows data from U.S. Bureau of Labor Statistics, Occupational Employment and Wage Statistics, May 2025 ("Hourly pay for hand laborers and material movers, May 2025") ` + ask},
			{Path: "html:#distance", Severity: audit.Warning, Message: `the page shows data from U.S. Census Bureau, Gazetteer files, 2024 ("Road miles to ten cities") ` + ask},
			{Path: "html:#typical-home", Severity: audit.Warning, Message: `the page shows data from U.S. Census Bureau, American Community Survey, 2020-2024 ("Homes by bedrooms, 2020-2024") ` + ask},
		}},
		{"a Dataset names each", audittest.Input{Draft: described, PageHTML: dataPage, PageURL: dataURL}, nil},
		{"a noindex page", audittest.Input{Draft: barePage, PageHTML: noindex(dataPage), PageURL: dataURL}, nil},
		{"no page", audittest.Input{Draft: barePage, PageURL: dataURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Dataset{}, tc.in, tc.want) })
	}
}
