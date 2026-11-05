package product

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// link is a link or button the page shows: the engagement checks look for the
// ones that invite a visitor to act.
type link struct {
	n     *xhtml.Node
	text  string // the element's text, trimmed
	label string // its aria-label
	href  string
	path  string // where an issue about it points
}

// firstLink is the first link or button of a page search engines index that
// the match accepts, with its document; nil when there is none or the page
// is known by its text.
func firstLink(c *audit.Context, match func(l link) bool) *link {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	var found *link
	webpage.Visit(c.Page.Doc, func(n *xhtml.Node) {
		if found != nil || (n.DataAtom != atom.A && n.DataAtom != atom.Button) {
			return
		}
		l := link{n: n, text: strings.TrimSpace(webpage.HTMLText(n))}
		l.label, _ = webpage.Attr(n, "aria-label")
		l.href, _ = webpage.Attr(n, "href")
		if match(l) {
			l.path = "html:" + webpage.SelectorFor(c.Page.Doc, n)
			found = &l
		}
	})
	return found
}
