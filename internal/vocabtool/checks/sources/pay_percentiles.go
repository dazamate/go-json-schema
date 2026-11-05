package sources

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(PayPercentiles{}) }

var payFor = regexp.MustCompile(`(?i)\bpay (?:for|of) ([^,(]+)`)

// PayPercentiles warns on a table of pay by percentile that no Occupation's
// estimatedSalary carries. A pay table's amounts are wages, not prices: it
// takes them, so the price checks leave them alone.
type PayPercentiles struct{}

// ID is the check's ID.
func (PayPercentiles) ID() string { return "sources/pay-percentiles" }

// Run reports each pay table, naming the job its caption gives and the area
// its first row heads.
func (PayPercentiles) Run(c *audit.Context) []audit.Issue {
	if jsonld.HasKey(c.Nodes, "estimatedSalary") {
		return nil
	}
	var out []audit.Issue
	for _, b := range dataBlocks(c) {
		if !b.pay() {
			continue
		}
		job := "the occupation"
		if m := payFor.FindStringSubmatch(b.caption); m != nil {
			job = strings.TrimSpace(m[1])
		}
		area := "the area"
		for _, tr := range webpage.QueryAll(b.n, "tr") {
			if ths := webpage.QueryAll(tr, "th"); len(ths) > 0 && len(webpage.QueryAll(tr, "td")) > 0 {
				area = webpage.HTMLText(ths[0])
				break
			}
		}
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, b.n), audit.Warning, "the page shows pay percentiles for %q in %s: add an Occupation (name, occupationLocation) with estimatedSalary as a MonetaryAmountDistribution (percentile10, percentile25, median, percentile75, percentile90, currency, unitText HOUR or YEAR)", job, area))
	}
	return out
}

// TakesPrices takes every amount in each pay table's block, carried or not.
func (PayPercentiles) TakesPrices(c *audit.Context, _ audit.Taken) audit.Taken {
	t := audit.Taken{}
	for _, b := range dataBlocks(c) {
		if b.pay() {
			t.Take(webpage.HTMLText(b.n))
		}
	}
	return t
}
