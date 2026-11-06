package costs

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(TaxMismatch{}) }

// TaxMismatch warns on a valueAddedTaxIncluded that says the opposite of
// what the page says of tax on its prices.
type TaxMismatch struct{}

// ID is the check's ID.
func (TaxMismatch) ID() string { return "costs/tax-mismatch" }

// Run reports each valueAddedTaxIncluded the page's wording contradicts.
func (TaxMismatch) Run(c *audit.Context) []audit.Issue {
	said, included := taxSaid(c)
	if said == "" {
		return nil
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		for _, v := range jsonld.Scalars(l.N, "valueAddedTaxIncluded") {
			if (v == "true") != included {
				out = append(out, audit.Issuef(l.Path+".valueAddedTaxIncluded", audit.Warning, "valueAddedTaxIncluded is %s but the page says %q", v, said))
			}
		}
	}
	return out
}
