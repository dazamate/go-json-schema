package content

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(RelatedProducts{}) }

// RelatedProducts warns when a block the page labels as related products
// lists product cards that the Product's isRelatedTo and isSimilarTo do not
// name. It needs the page's document, on a page search engines index.
type RelatedProducts struct{}

// ID is the check's ID.
func (RelatedProducts) ID() string { return "content/related-products" }

// Run reports each such block.
func (RelatedProducts) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range relatedOfKind(c, relatedProducts) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's %q block lists products (%s) that the Product's isRelatedTo and isSimilarTo do not name: add each as a Product (name, url, image, offers)", f.label, strings.Join(f.names, ", ")))
	}
	return out
}
