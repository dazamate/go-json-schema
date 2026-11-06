// Package costs holds the price checks. On a site that sells nothing, its
// prices are what others charge: a rate is a UnitPriceSpecification, a total
// broken into parts a CompoundPriceSpecification, a cost or estimate a
// MonetaryAmount and goods priced across sellers a Product with an
// AggregateOffer; an Offer is only for what the business itself sells. A
// shop's prices stay its Offers. Whatever price no check reports, the prices
// checks ask the markup to carry, with its currency and its tax.
package costs

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// priceSpan is a price or a range of prices: "$380", "$380–$960".
const priceSpan = `(` + txt.PriceText + `(?:\s*(?:–|-|to)\s*` + txt.PriceText + `)?)`

// costPage reports a page search engines index on a site that sells
// nothing: there its prices are what others charge.
func costPage(c *audit.Context) bool { return c.Indexed() && !c.Page.Sells() }

// priceKeys are the properties that carry a node's price: the price
// properties, and on a MonetaryAmount or PriceSpecification its values. The
// first five are the prices of things for sale.
func priceKeys(c *audit.Context, n *jsonld.Object) []string {
	keys := []string{"price", "lowPrice", "highPrice", "minPrice", "maxPrice"}
	if c.Is(n, "MonetaryAmount") || c.Is(n, "PriceSpecification") {
		keys = append(keys, "value", "minValue", "maxValue")
	}
	return keys
}

// carried are the amounts the draft carries.
type carried struct {
	all  audit.Taken // every amount a price property or a monetary value holds
	sold audit.Taken // the amounts a price property holds: the prices of things for sale, which tax wording is about (a shipping rate's MonetaryAmount is not one)
}

// carriedKey is the memo key of the draft's carried amounts.
type carriedKey struct{}

// carriedBy is the amounts the draft carries, worked out once. The sets are
// shared: read them only.
func carriedBy(c *audit.Context) carried {
	return audit.Memo(c, carriedKey{}, func() carried {
		out := carried{all: audit.Taken{}, sold: audit.Taken{}}
		for _, l := range c.Nodes {
			for i, k := range priceKeys(c, l.N) {
				for _, s := range jsonld.Scalars(l.N, k) {
					if f, ok := txt.Amount(s); ok {
						out.all[f] = true
						if i < 5 {
							out.sold[f] = true
						}
					}
				}
			}
		}
		return out
	})
}

// fresh reports text holding an amount the draft does not carry and that
// none of the taken sets holds: the cost checks report only those.
func fresh(c *audit.Context, text string, taken ...audit.Taken) bool {
	marked := carriedBy(c).all
	return slices.ContainsFunc(txt.PriceAmount.FindAllStringSubmatch(text, -1), func(m []string) bool {
		f, ok := txt.PriceOf(m)
		return ok && !marked[f] && !slices.ContainsFunc(taken, func(t audit.Taken) bool { return t[f] })
	})
}

// clause is the text before position i in s back to the start of its
// sentence, at most n bytes.
func clause(s string, i, n int) string {
	from := max(0, i-n)
	t := s[from:i]
	if k := strings.LastIndex(t, ". "); k >= 0 {
		t = t[k+2:]
	}
	return t
}
