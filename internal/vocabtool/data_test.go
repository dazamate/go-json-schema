package vocabtool

import "testing"

// dataPage has two data modules with their sources, one of them a pay table,
// and a price table with no statistical source.
const dataPage = `<!doctype html><html><body><main><h1>Moving to San Diego</h1>
<section id="movers-nearby"><h2>How many movers work in the county</h2>
<p>150 moving companies with paid employees in San Diego County.</p>
<figure id="rate"><figcaption>Moving companies per 100,000 residents, 2023</figcaption><svg></svg></figure>
<details><summary>See the numbers</summary><table><caption>Moving companies by area</caption><tr><th>Area</th><th>Companies</th></tr><tr><th>San Diego County</th><td>150</td></tr></table></details>
<p><small>Source: <cite>U.S. Census Bureau, County Business Patterns, 2023: San Diego County</cite>.</small></p></section>
<section id="mover-pay"><h2>Hourly pay for San Diego's crews</h2>
<p>Movers in the San Diego-Chula Vista-Carlsbad, CA metro area earn $18 to $23 an hour.</p>
<table><caption>Hourly pay for hand laborers and material movers, May 2025</caption>
<tr><th>Area</th><th>Lowest-paid tenth earn under</th><th>Middle half</th><th>Typical</th><th>Highest-paid tenth earn over</th></tr>
<tr><th>San Diego-Chula Vista-Carlsbad, CA metro area</th><td>$17.51</td><td>$18.14–$22.86</td><td>$20.83</td><td>$27.88</td></tr></table>
<p><small>Source: <cite>U.S. Bureau of Labor Statistics, Occupational Employment and Wage Statistics, May 2025</cite>.</small></p></section>
<section id="distance"><h2>Road miles from San Diego</h2><p>To Las Vegas, the nearest of these cities, a 2-bedroom move runs $1,500–$3,550.</p>
<table><caption>Road miles to ten cities</caption><tr><th>City</th><th>Miles</th></tr><tr><th>Las Vegas</th><td>331</td></tr></table>
<p><small>Source: <cite>U.S. Census Bureau, Gazetteer files, 2024</cite>.</small></p></section>
<section id="typical-home"><h2>The usual home</h2><p>The most common home in San Diego has 3 bedrooms.</p>
<figure><figcaption>Homes by bedrooms, 2020-2024</figcaption><svg></svg></figure>
<p><small>Source: <cite>U.S. Census Bureau, American Community Survey, 2020-2024</cite>.</small></p></section>
<section id="prices"><h2>Truck prices</h2><table><caption>Truck rental prices</caption><tr><th>Truck</th><th>Price</th></tr><tr><td>10 ft</td><td>$30</td></tr></table>
<p>Source: <cite>U-Haul</cite></p></section>
</main></body></html>`

func TestCheckFindsTheDataAPageShows(t *testing.T) {
	bare := `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/san-diego/","name":"Moving to San Diego"}`
	r := checkSite(t, bare, dataPage, "https://movers.test/san-diego/")
	expectWarnings(t, r, [][]string{
		{`data from U.S. Census Bureau, County Business Patterns, 2023 ("Moving companies per 100,000 residents, 2023")`, "no Dataset describes it", "variableMeasured"},
		{`states "150 moving companies with paid employees in San Diego County."`, "add an Observation", "StatisticalVariable"},
		{`states "The most common home in San Diego has 3 bedrooms."`, "add an Observation", "FloorPlan", "numberOfBedrooms"},
		{`pay percentiles for "hand laborers and material movers" in San Diego-Chula Vista-Carlsbad, CA metro area`, "estimatedSalary", "MonetaryAmountDistribution"},
	})
	if n := issueCount(r, "no Dataset describes it"); n != 4 {
		t.Errorf("%d Dataset asks, want the four modules with a statistical source: %+v", n, r.Issues)
	}
	if warns(r, "add an Observation", "$1,500") {
		t.Errorf("a price estimate read as a statistic: %+v", r.Issues)
	}
	// wages are not prices
	for _, w := range []string{"$17.51", "$18 to $23"} {
		if n := issueCount(r, w); n != 0 {
			t.Errorf("%s reported as a price: %+v", w, r.Issues)
		}
	}
	marked := `{"@context":"https://schema.org","@graph":[` + bare + `,
		{"@type":"Dataset","name":"Moving companies per 100,000 residents, 2023","creator":{"@type":"GovernmentOrganization","name":"U.S. Census Bureau"},
			"mainEntity":{"@type":"Observation","value":150,"observationAbout":{"@type":"AdministrativeArea","name":"San Diego County"}}},
		{"@type":"Dataset","name":"Hourly pay for hand laborers and material movers, May 2025"},
		{"@type":"Dataset","name":"Road miles to ten cities"},
		{"@type":"Dataset","name":"Homes by bedrooms, 2020-2024","mainEntity":{"@type":"Observation","value":3}},
		{"@type":"Occupation","name":"Hand laborers and material movers","estimatedSalary":{"@type":"MonetaryAmountDistribution","median":20.83,"currency":"USD"}}]}`
	r = checkSite(t, marked, dataPage, "https://movers.test/san-diego/")
	for _, w := range []string{"no Dataset describes it", "add an Observation", "estimatedSalary"} {
		if n := issueCount(r, w); n != 0 {
			t.Errorf("%q with the data marked up: %+v", w, r.Issues)
		}
	}
}
