package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(RelatedGuide{}) }

// RelatedGuide warns when a block the page labels as a guide's own pages
// ("In this guide") links to three or more pages of the site that no ItemList
// in the draft lists. It needs the page's document, on a page search engines
// index.
type RelatedGuide struct{}

// ID is the check's ID.
func (RelatedGuide) ID() string { return "content/related-guide" }

// Run reports each such block.
func (RelatedGuide) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range relatedOfKind(c, relatedGuide) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's %q block lists %d pages of the site: add an ItemList of them (a ListItem for each with position, url and name) to the WebPage's hasPart; a hub page whose purpose is the list is a CollectionPage with the ItemList as its mainEntity", f.label, f.count))
	}
	return out
}
