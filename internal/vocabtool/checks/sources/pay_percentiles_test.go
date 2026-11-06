package sources_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
)

// dataPage has four data modules with their sources, one of them a pay
// table, and a price table with no statistical source.
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

const (
	dataURL  = "https://movers.test/san-diego/"
	barePage = `{"@context":"https://schema.org","@type":"WebPage","url":"https://movers.test/san-diego/","name":"Moving to San Diego"}`
)

// percentilePage is electriciansalary.com's state table (trimmed): P10 to P90
// heads, no caption, no source line, areas in plain cells.
const percentilePage = `<!doctype html><html><body><main><h1>Where Does Your Electrician Pay Rank?</h1>
<section class="py-8"><h2>Published percentile anchors, state by state</h2><p>The five values the tool reads.</p>
<div class="schematic-table"><table><thead><tr><th>Area</th><th>P10</th><th>P25</th><th>Median</th><th>P75</th><th>P90</th><th>P90 minus P10</th></tr></thead><tbody>
<tr><td>United States</td><td>$42,640</td><td>$49,430</td><td>$63,190</td><td>$83,940</td><td>$108,510</td><td>$65,870</td></tr>
<tr><td>Oregon</td><td>$59,550</td><td>$79,110</td><td>$101,310</td><td>$121,680</td><td>$131,530</td><td>$71,980</td></tr></tbody></table></div></section>
<section><h2>Plans</h2><table><thead><tr><th>Plan</th><th>Basic</th><th>Median</th></tr></thead><tbody><tr><td>Solo</td><td>$9</td><td>$19</td></tr></tbody></table></section></main></body></html>`

func TestPayPercentiles(t *testing.T) {
	paid := `{"@context":"https://schema.org","@type":"Occupation","name":"Hand laborers and material movers","estimatedSalary":{"@type":"MonetaryAmountDistribution","median":20.83,"currency":"USD"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a pay table", audittest.Input{Draft: barePage, PageHTML: dataPage, PageURL: dataURL}, []audit.Issue{
			{Path: "html:#mover-pay", Severity: audit.Warning, Message: `the page shows pay percentiles for "hand laborers and material movers" in San Diego-Chula Vista-Carlsbad, CA metro area: add an Occupation (name, occupationLocation) with estimatedSalary as a MonetaryAmountDistribution (percentile10, percentile25, median, percentile75, percentile90, currency, unitText HOUR or YEAR)`},
		}},
		{"a table headed by percentiles", audittest.Input{Draft: barePage, PageHTML: percentilePage, PageURL: "https://pay.test/electrician/"}, []audit.Issue{
			{Path: "html:body > main > section:nth-of-type(1) > div > table", Severity: audit.Warning, Message: `the page shows pay percentiles for "the occupation" in United States: add an Occupation (name, occupationLocation) with estimatedSalary as a MonetaryAmountDistribution (percentile10, percentile25, median, percentile75, percentile90, currency, unitText HOUR or YEAR)`},
		}},
		{"an estimatedSalary carries it", audittest.Input{Draft: paid, PageHTML: dataPage, PageURL: dataURL}, nil},
		{"a noindex page", audittest.Input{Draft: barePage, PageHTML: audittest.Noindex(dataPage), PageURL: dataURL}, nil},
		{"no page", audittest.Input{Draft: barePage, PageURL: dataURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, sources.PayPercentiles{}, tc.in, tc.want) })
	}
}
