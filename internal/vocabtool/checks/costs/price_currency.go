package costs

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(PriceCurrency{}) }

// PriceCurrency warns on a node that has a price but no currency: an amount
// alone says nothing of what it costs.
type PriceCurrency struct{}

// ID is the check's ID.
func (PriceCurrency) ID() string { return "costs/price-currency" }

// Run reports each priced node with neither priceCurrency nor currency.
func (PriceCurrency) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		priced := slices.ContainsFunc(priceKeys(c, l.N), func(k string) bool {
			return slices.ContainsFunc(jsonld.Scalars(l.N, k), func(s string) bool { _, ok := txt.Amount(s); return ok })
		})
		if priced && len(jsonld.Strs(l.N, "priceCurrency"))+len(jsonld.Strs(l.N, "currency")) == 0 {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "this %s has a price but no priceCurrency: set it (AUD, USD, EUR)", nodeNoun(l.N)))
		}
	}
	return out
}

// nodeNoun is the node's first type, or "node" for one that has none.
func nodeNoun(n *jsonld.Object) string {
	if types := jsonld.TypeNames(n); len(types) > 0 {
		return types[0]
	}
	return "node"
}
