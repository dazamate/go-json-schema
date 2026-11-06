package content

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// searched is what the page shows of a search that the draft has no
// SearchAction for: a search box, or else a link to the site's search page.
// The page gets the one finding, the box before the link.
type searched struct {
	box  *xhtml.Node // the first search box that is shown
	link *xhtml.Node // the first link to the search page, when there is no box
	to   string      // the link's address
}

// searchKey is the memo key of the page's search.
type searchKey struct{}

// searchOf is the search the page shows and the draft lacks a SearchAction
// for. The two search checks share it. It needs the page's document.
func searchOf(c *audit.Context) searched {
	return audit.Memo(c, searchKey{}, func() searched {
		p := c.Page
		if p.Doc == nil || slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "SearchAction") }) {
			return searched{}
		}
		for _, n := range webpage.QueryAll(p.Doc, `input[type="search"], [role="search"] input, input[role="searchbox"]`) {
			if webpage.Shown(n) {
				return searched{box: n}
			}
		}
		// a text field named like a search, alone in its form, is a search box too
		for _, n := range webpage.QueryAll(p.Doc, `input[name="q" i], input[name="s" i], input[name="search" i], input[name="query" i], input[name="keyword" i], input[name="keywords" i], input[name="search_query" i], input[name="searchterm" i]`) {
			t, _ := webpage.Attr(n, "type")
			if !webpage.Shown(n) || (t != "" && !strings.EqualFold(t, "text")) {
				continue
			}
			form := n.Parent
			for form != nil && form.DataAtom != atom.Form {
				form = form.Parent
			}
			if form != nil && len(entryControls(form)) <= 1 && len(webpage.QueryAll(form, "textarea")) == 0 {
				return searched{box: n}
			}
		}
		for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
			u, ok := p.SiteLink(a)
			label, _ := webpage.Attr(a, "aria-label")
			if ok && webpage.Shown(a) && slices.Contains(webpage.Segments(webpage.MustParse(u)), "search") && strings.Contains(strings.ToLower(label+" "+webpage.HTMLText(a)), "search") {
				return searched{link: a, to: u}
			}
		}
		return searched{}
	})
}
