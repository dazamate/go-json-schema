package shop

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// kindPageType is the WebPage type each kind of page takes.
var kindPageType = map[webpage.Kind]string{
	webpage.KindProduct: "ItemPage", webpage.KindListing: "CollectionPage", webpage.KindSearch: "SearchResultsPage",
	webpage.KindContact: "ContactPage", webpage.KindAbout: "AboutPage", webpage.KindFAQ: "FAQPage", webpage.KindCheckout: "CheckoutPage",
}

// pageNode is the WebPage type a page calls for and the draft's node for the
// page.
type pageNode struct {
	want string      // the type the page's kind calls for, or WebPage
	node jsonld.Node // the draft's first WebPage with properties; N is nil when it has none
}

// pageNodeKey is the memo key of the page's WebPage type and node.
type pageNodeKey struct{}

// pageNodeOf is the page's WebPage type and node, worked out once. It is nil
// on a page search engines skip, or one known only by its text: these
// checks read the page's document.
func pageNodeOf(c *audit.Context) *pageNode {
	return audit.Memo(c, pageNodeKey{}, func() *pageNode {
		if !c.Indexed() || c.Page.Doc == nil {
			return nil
		}
		p := &pageNode{want: kindPageType[c.Page.Kind]}
		if p.want == "" {
			p.want = "WebPage"
		}
		if i := slices.IndexFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "WebPage") && len(l.N.Keys) > 1 }); i >= 0 {
			p.node = c.Nodes[i]
		}
		return p
	})
}
