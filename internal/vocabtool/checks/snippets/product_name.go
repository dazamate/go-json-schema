package snippets

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(ProductName{}) }

// ProductName warns about a Product with no name: Search Console marks the
// item invalid without one, and a snippet shows the product by it.
type ProductName struct{}

// ID is the check's ID.
func (ProductName) ID() string { return "snippets/product-name" }

// Run reports each Product with no name.
func (ProductName) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, p := range productsOf(c) {
		if !has(p.N, "name") {
			out = append(out, audit.Issuef(p.Path, audit.Warning, critical+`Missing field "name": give this %s the name the page shows for it`, p.ty))
		}
	}
	return out
}
