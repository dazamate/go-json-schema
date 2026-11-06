package content

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(RelatedLinks{}) }

// RelatedLinks warns when a block the page labels as related reading links
// to pages of the site that the WebPage's relatedLink does not list. A block
// of product cards or a guide's own pages is another check's. It needs the
// page's document, on a page search engines index.
type RelatedLinks struct{}

// ID is the check's ID.
func (RelatedLinks) ID() string { return "content/related-links" }

// Run reports each such block.
func (RelatedLinks) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range relatedOfKind(c, relatedLinks) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's %q links are not in the WebPage's relatedLink: add %s", f.label, strings.Join(f.links, ", ")))
	}
	return out
}
