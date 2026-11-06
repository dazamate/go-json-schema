package shop

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(WebSite{}) }

// WebSite warns when the draft has no WebSite: every page of a site carries
// the site's WebSite, which the page's WebPage is part of and the business
// publishes. It needs the page's document, on a page search engines index.
type WebSite struct{}

// ID is the check's ID.
func (WebSite) ID() string { return "shop/web-site" }

// Run reports a draft with no WebSite, naming the node to add.
func (WebSite) Run(c *audit.Context) []audit.Issue {
	s := sitewideOf(c)
	if s == nil || s.site != nil {
		return nil
	}
	home, id := "the site's home page", "the home page's URL + #website"
	if c.Page.URL != nil {
		home = c.Page.URL.Scheme + "://" + c.Page.URL.Host + "/"
		id = home + "#website"
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the draft has no WebSite: add the site's WebSite to every page (@id %s, url %s, name, inLanguage %s, publisher the business) and set the WebPage's isPartOf to it", id, home, txt.OrDefault(s.lang, "the page's language"))}
}
