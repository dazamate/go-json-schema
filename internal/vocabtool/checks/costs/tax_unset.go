package costs

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(TaxUnset{}) }

// TaxUnset warns when the page says whether its prices include tax and no
// PriceSpecification sets valueAddedTaxIncluded.
type TaxUnset struct{}

// ID is the check's ID.
func (TaxUnset) ID() string { return "costs/tax-unset" }

// Run reports the page's tax wording when no node has valueAddedTaxIncluded.
func (TaxUnset) Run(c *audit.Context) []audit.Issue {
	said, included := taxSaid(c)
	if said == "" || slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool { return len(jsonld.Scalars(l.N, "valueAddedTaxIncluded")) > 0 }) {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page's prices are %q but no PriceSpecification sets valueAddedTaxIncluded: set it %t", said, included)}
}
