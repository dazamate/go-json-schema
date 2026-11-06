package product

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(SaleItems{}) }

// SaleItems warns on the was prices of a shop's sale items that no
// StrikethroughPrice carries: several on one page, or one on a page that is
// no product's. WasPrice takes them.
type SaleItems struct{}

// ID is the check's ID.
func (SaleItems) ID() string { return "product/sale-items" }

// Run reports the was prices the draft lacks, unless WasPrice reports the
// product page's one.
func (SaleItems) Run(c *audit.Context) []audit.Issue {
	s, ok := saleOf(c)
	if !ok || len(s.missing) == 0 || (len(s.missing) == 1 && c.Page.Kind == webpage.KindProduct) {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows was prices (%s) on sale items: give each item's Offer a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice and its was price", strings.Join(s.missing, ", "))}
}
