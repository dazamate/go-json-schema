package shop

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(WebPage{}) }

// WebPage warns when the draft has no WebPage for the page, naming the type
// its kind of page takes and what the node should hold: a page's structured
// data hangs off its WebPage. It needs the page's document, on a page search
// engines index.
type WebPage struct{}

// ID is the check's ID.
func (WebPage) ID() string { return "shop/web-page" }

// Run reports a page with no WebPage node.
func (WebPage) Run(c *audit.Context) []audit.Issue {
	p := pageNodeOf(c)
	if p == nil || p.node.N != nil {
		return nil
	}
	extra := map[webpage.Kind]string{
		webpage.KindProduct: "mainEntity the Product, breadcrumb the BreadcrumbList",
		webpage.KindListing: "mainEntity the ItemList of its products, breadcrumb the BreadcrumbList",
		webpage.KindHome:    "about and mainEntity the business",
		webpage.KindContact: "mainEntity the business",
		webpage.KindAbout:   "mainEntity the business",
	}[c.Page.Kind]
	if extra == "" {
		extra = "breadcrumb the BreadcrumbList when it has one"
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page has no WebPage: add a %s for it (url, name, isPartOf the WebSite, %s)", p.want, extra)}
}
