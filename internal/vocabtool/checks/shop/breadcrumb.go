package shop

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(Breadcrumb{}) }

// Breadcrumb warns when the page has a breadcrumb nav, the draft has its
// BreadcrumbList, and the page's WebPage does not name it as its breadcrumb.
// Like the check of isPartOf, it waits for the draft to have a WebSite.
type Breadcrumb struct{}

// ID is the check's ID.
func (Breadcrumb) ID() string { return "shop/breadcrumb" }

// Run reports the WebPage, when the page's breadcrumb nav has a
// BreadcrumbList in the draft the WebPage leaves out.
func (Breadcrumb) Run(c *audit.Context) []audit.Issue {
	s := sitewideOf(c)
	if s == nil || s.site == nil || s.page == nil || !s.crumbs || !c.Has("BreadcrumbList") {
		return nil
	}
	if _, ok := s.page.N.Get("breadcrumb"); ok {
		return nil
	}
	return []audit.Issue{audit.Issuef(s.page.Path, audit.Warning, "the WebPage has no breadcrumb: set it to the BreadcrumbList of the page's breadcrumb nav")}
}
