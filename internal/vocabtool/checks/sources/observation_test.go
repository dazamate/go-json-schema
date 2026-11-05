package sources_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

func TestObservation(t *testing.T) {
	observed := `{"@context":"https://schema.org","@graph":[
		{"@type":"Dataset","name":"Moving companies per 100,000 residents, 2023","mainEntity":{"@type":"Observation","value":150}},
		{"@type":"Dataset","name":"Homes by bedrooms, 2020-2024","mainEntity":{"@type":"Observation","value":3}}]}`
	// a ratio's sentence leads with its base, which an Observation named by
	// the sentence carries
	tulsa := `<!doctype html><html><body><main><h1>Moving in Tulsa</h1>
<section id="place-moves"><h2>How often Tulsans move</h2>
<p>1 in 6 people in Tulsa moved in the last year</p>
<figure><figcaption>People who moved in the last year, as a share of residents, 2020-2024</figcaption><svg></svg></figure>
<p><small>Source: <cite>U.S. Census Bureau, American Community Survey, 2020-2024</cite>.</small></p></section>
</main></body></html>`
	named := `{"@context":"https://schema.org","@type":"Observation","name":"1 in 6 people in  Tulsa moved in the last year","value":17,"unitCode":"P1"}`
	ask := "add an Observation (observationAbout the place, variableMeasured a StatisticalVariable with measuredProperty and statType, value, observationDate) to the Dataset"
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"lead figures", audittest.Input{Draft: barePage, PageHTML: dataPage, PageURL: dataURL}, []audit.Issue{
			{Path: "html:#movers-nearby", Severity: audit.Warning, Message: `the page states "150 moving companies with paid employees in San Diego County." from U.S. Census Bureau, County Business Patterns, 2023: ` + ask},
			{Path: "html:#typical-home", Severity: audit.Warning, Message: `the page states "The most common home in San Diego has 3 bedrooms." from U.S. Census Bureau, American Community Survey, 2020-2024: ` + ask + "; a FloorPlan (numberOfBedrooms, numberOfRooms) describes the home it counts"},
		}},
		{"a ratio no Observation names", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Observation","value":17,"unitCode":"P1"}`, PageHTML: tulsa, PageURL: "https://movers.test/tulsa/"}, []audit.Issue{
			{Path: "html:#place-moves", Severity: audit.Warning, Message: `the page states "1 in 6 people in Tulsa moved in the last year" from U.S. Census Bureau, American Community Survey, 2020-2024: ` + ask},
		}},
		{"a year is no lead figure", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"Observation","value":4.2,"unitCode":"MIL"}`, PageHTML: strings.Replace(tulsa, "1 in 6 people in Tulsa moved in the last year", "In 2023, 4.2 million households moved.", 1), PageURL: "https://movers.test/tulsa/"}, nil},
		{"Observations carry the figures", audittest.Input{Draft: observed, PageHTML: dataPage, PageURL: dataURL}, nil},
		{"an Observation named by the sentence", audittest.Input{Draft: named, PageHTML: tulsa, PageURL: "https://movers.test/tulsa/"}, nil},
		{"a noindex page", audittest.Input{Draft: barePage, PageHTML: audittest.Noindex(dataPage), PageURL: dataURL}, nil},
		{"no page", audittest.Input{Draft: barePage, PageURL: dataURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.Observation{}, tc.in, tc.want) })
	}
}
