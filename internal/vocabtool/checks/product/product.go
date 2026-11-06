package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Product{}) }

// Product warns when a product page has no Product node: a product page's
// structured data is its Product, with the Offer that sells it.
type Product struct{}

// ID is the check's ID.
func (Product) ID() string { return "product/product" }

// Run reports a product page whose draft describes no Product.
func (Product) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || len(productsOf(c)) > 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page sells %q but no Product describes it: add a Product as the ItemPage's mainEntity (name, image, description, sku, brand a Brand, offers an Offer with url, price, priceCurrency, availability, itemCondition and seller the business)", txt.Clip(c.Page.H1(), 80))}
}
