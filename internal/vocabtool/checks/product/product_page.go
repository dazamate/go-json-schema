package product

import (
	"regexp"
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// onProductPage reports a product page that search engines index: the page
// a product's own checks read.
func onProductPage(c *audit.Context) bool {
	return c.Indexed() && c.Page.Kind == webpage.KindProduct && !onListing(c.Page)
}

var listingSegments = regexp.MustCompile(`^(collections?|categor(y|ies))$`)

// onListing reports a page the kind rules took for a product's (a cart
// drawer's add-to-cart form on /collections/sale) whose address is a shop's
// collection or category and names no product: /collections/rings/products/opal
// is a product's.
func onListing(p *webpage.Facts) bool {
	segs := webpage.Segments(p.URL)
	return slices.ContainsFunc(segs, listingSegments.MatchString) &&
		!slices.ContainsFunc(segs, func(s string) bool { return s == "product" || s == "products" })
}

// productsKey is the memo key of the draft's Product nodes.
type productsKey struct{}

// productsOf are the draft's Product nodes with properties, ProductGroups and
// other subtypes included, found once.
func productsOf(c *audit.Context) []jsonld.Node {
	return audit.Memo(c, productsKey{}, func() []jsonld.Node {
		var out []jsonld.Node
		for _, l := range c.Nodes {
			if c.Is(l.N, "Product") && len(l.N.Keys) > 1 {
				out = append(out, l)
			}
		}
		return out
	})
}

// headKey is the memo key of a product page's head.
type headKey struct{}

// headOf is the product page's name and title, joined: where a set, an
// audience or a kind of product is announced.
func headOf(c *audit.Context) string {
	return audit.Memo(c, headKey{}, func() string {
		return c.Page.H1() + " · " + titleOf(c.Page)
	})
}

// titleOf is the page's title element.
func titleOf(p *webpage.Facts) string {
	if p.Doc == nil {
		return ""
	}
	if ts := webpage.QueryAll(p.Doc, "title"); len(ts) > 0 {
		return webpage.HTMLText(ts[0])
	}
	return ""
}
