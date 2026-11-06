package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(AddOn{}) }

// AddOn warns on a paid extra a shop's page offers (gift wrapping,
// engraving) when no Offer has an addOn. UnitPrice takes its price.
type AddOn struct{}

// ID is the check's ID.
func (AddOn) ID() string { return "product/add-on" }

// Run reports the first paid extra the page offers.
func (AddOn) Run(c *audit.Context) []audit.Issue {
	_, extra := extras(c)
	if extra == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page offers %q as an extra: add it as the Offer's addOn, an Offer (name, price, priceCurrency)", extra)}
}
