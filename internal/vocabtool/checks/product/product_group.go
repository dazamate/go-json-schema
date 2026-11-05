package product

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ProductGroup{}) }

// ProductGroup warns when a product page offers a choice of size, colour and
// the like and the draft has no ProductGroup for it: a product in several
// variants is a ProductGroup whose variants each have their own Offer.
type ProductGroup struct{}

// ID is the check's ID.
func (ProductGroup) ID() string { return "product/product-group" }

// Run reports the choices the page offers, with what the group varies by.
func (ProductGroup) Run(c *audit.Context) []audit.Issue {
	cs := choicesOf(c)
	if cs == nil || cs.group != nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page offers a choice of %s: make the product a ProductGroup (productGroupID, variesBy %s) with a Product for each choice in hasVariant, each with its %s and its own Offer", strings.Join(cs.said, " and "), strings.Join(cs.urls, " and "), strings.Join(slices.Compact(slices.Clone(cs.props)), " and "))}
}
