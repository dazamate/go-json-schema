package shop

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// sitewide is the draft's WebSite and the WebPage that should be part of it,
// with what the page says of its language and breadcrumb.
type sitewide struct {
	lang   string       // the page's language, from its html element, or ""
	site   *jsonld.Node // the draft's first WebSite with properties, or nil
	page   *jsonld.Node // the draft's WebPage for the page, or nil
	crumbs bool         // the page has a breadcrumb nav
}

// sitewideKey is the memo key of the page's WebSite and WebPage.
type sitewideKey struct{}

// sitewideOf is the draft's WebSite and main WebPage, found once for the
// checks that share them. It is nil on a page search engines skip, or one
// known only by its text: these checks read the page's document.
func sitewideOf(c *audit.Context) *sitewide {
	return audit.Memo(c, sitewideKey{}, func() *sitewide {
		if !c.Indexed() || c.Page.Doc == nil {
			return nil
		}
		p := c.Page
		s := &sitewide{}
		for _, h := range webpage.QueryAll(p.Doc, "html[lang]") {
			s.lang, _ = webpage.Attr(h, "lang")
		}
		if i := slices.IndexFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "WebSite") && len(l.N.Keys) > 1 }); i >= 0 {
			s.site = &c.Nodes[i]
		}
		if wp, ok := audit.MainWebPage(c.Vocab, c.Nodes, p); ok {
			s.page = &wp
		}
		// the breadcrumb nav's BreadcrumbList belongs to the page
		s.crumbs = slices.ContainsFunc(webpage.PageSections(p.Doc), func(s *webpage.Section) bool { return s.Type == "BreadcrumbList" })
		return s
	})
}
