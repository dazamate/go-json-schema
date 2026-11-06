package product

import (
	"regexp"
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(LowStock{}) }

var lowStock = regexp.MustCompile(`(?i)\bonly (\d+) left(?: in stock)?\b|\b(\d+) left in stock\b|\bonly (\d+) in stock\b`)

// LowStock warns when a shop's page says how few of a product are left ("Only
// 3 left") and no Offer carries an inventoryLevel for it or a
// LimitedAvailability.
type LowStock struct{}

// ID is the check's ID.
func (LowStock) ID() string { return "product/low-stock" }

// Run reports the stock levels the page states, once, up to four.
func (LowStock) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	levels := jsonld.Under(c.Nodes, "inventoryLevel")
	var low []string
	for _, m := range lowStock.FindAllStringSubmatch(c.Page.Main, -1) {
		f, _ := txt.Amount(txt.FirstNonEmpty(m[1:]...))
		if !txt.HoldsNumber(levels, f) && !jsonld.KeyMember(c.Nodes, "availability", "LimitedAvailability") && !slices.Contains(low, m[0]) {
			low = append(low, m[0])
		}
	}
	if len(low) == 0 {
		return nil
	}
	if len(low) > 4 {
		low = append(low[:4:4], "…")
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %s: set each such product's Offer inventoryLevel (a QuantitativeValue with the number left as value) and availability https://schema.org/LimitedAvailability", txt.QuoteAll(low))}
}
