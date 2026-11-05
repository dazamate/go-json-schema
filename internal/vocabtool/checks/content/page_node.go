package content

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// pageNode is the path of the WebPage the draft describes the page with: the
// one whose url is the canonical URL, or else the first WebPage. A WebPage
// that is no more than a reference does not count. It is "$" for a draft with
// no WebPage.
func pageNode(c *audit.Context) string {
	first := ""
	for _, l := range c.Nodes {
		if !c.Is(l.N, "WebPage") || len(l.N.Keys) < 2 {
			continue
		}
		if c.Page.Base != nil && slices.ContainsFunc(jsonld.Strs(l.N, "url"), func(s string) bool { return txt.NormLink(s) == txt.NormURL(c.Page.Base) }) {
			return l.Path
		}
		if first == "" {
			first = l.Path
		}
	}
	if first == "" {
		return "$"
	}
	return first
}
