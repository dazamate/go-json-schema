package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

// outlookPage is the BLS Occupational Outlook Handbook's job outlook (trimmed): the
// chart's label leads with a year range, and its Source line ends a sentence
// with the lead text of the next paragraph after it.
const outlookPage = `<!doctype html><html><body><main><h1>Electricians</h1><article aria-labelledby="job-outlook"><h2 id="job-outlook"><span> Job Outlook</span></h2>
<div class="ooh-chart"><h3> Electricians</h3><p> Percent change in employment, projected 2025–35</p><dl><dt><span> Electricians </span></dt><dd>11%</dd></dl><p class="footer"> Note: <br>Data are from the Occupational Employment and Wage Statistics program.<br> All Occupations includes all occupations in the U.S. Economy.<br>Source: U.S. Bureau of Labor Statistics.</p></div>
<p> Employment of electricians is projected to grow 9 percent from 2025 to 2035, much faster than the average for all occupations.</p>
<table class="regular" id="outlook-table"><caption><span class="tableTitle"> Employment projections data for electricians, 2025–35</span></caption><thead><tr><th>Occupational Title</th><th>Employment, 2025</th></tr></thead><tbody><tr><td>Electricians</td><td>757,200</td></tr></tbody></table></article></main></body></html>`

// bulletinPage is an ONS bulletin (trimmed): a chart with a numbered heading, no
// figcaption, and its source in a heading under it.
const bulletinPage = `<!doctype html><html><body><main><h1>House Price Index, UK</h1><div class="content"><p>Average UK house prices fell.</p>
<div class="markdown-chart-container"><h3>Figure 1: Average UK house prices fell in the 12 months to December 2023 (provisional estimate)</h3><div class="chart"><span>10</span><span>15</span></div>
<h4>Source: UK House Price Index from the HM Land Registry, Registers of Scotland, Land and Property Services Northern Ireland, Office for National Statistics</h4></div>
<div class="markdown-chart-container"><h3>Figure 2: Prices by region</h3><div class="chart"></div><h4>Source: UK House Price Index from the HM Land Registry</h4></div></div></main></body></html>`

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
		{"a source line that ends its sentence", audittest.Input{Draft: barePage, PageHTML: outlookPage, PageURL: "https://jobs.test/electricians/"}, []audit.Issue{
			{Path: "html:body > main > article", Severity: audit.Warning, Message: `the page shows data from U.S. Bureau of Labor Statistics ("Employment projections data for electricians, 2025–35") ` + ask},
		}},
		{"a numbered chart title and its source heading", audittest.Input{Draft: barePage, PageHTML: bulletinPage, PageURL: "https://ons.test/hpi/"}, []audit.Issue{
			{Path: "html:body > main > div > div:nth-of-type(1)", Severity: audit.Warning, Message: `the page shows data from UK House Price Index from the HM Land Registry, Registers of Scotland, Land and Property Services No… ("Figure 1: Average UK house prices fell in the 12 months to December 2023 (provis…") ` + ask},
		}},
		{"a Dataset names each", audittest.Input{Draft: described, PageHTML: dataPage, PageURL: dataURL}, nil},
		{"a noindex page", audittest.Input{Draft: barePage, PageHTML: audittest.Noindex(dataPage), PageURL: dataURL}, nil},
		{"no page", audittest.Input{Draft: barePage, PageURL: dataURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Dataset{}, tc.in, tc.want) })
	}
}
