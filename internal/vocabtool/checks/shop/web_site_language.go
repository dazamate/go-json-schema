package shop

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(WebSiteLanguage{}) }

// WebSiteLanguage warns when the draft's WebSite has no inLanguage and the
// page's html element names its language.
type WebSiteLanguage struct{}

// ID is the check's ID.
func (WebSiteLanguage) ID() string { return "shop/web-site-language" }

// Run reports the WebSite, naming the page's language.
func (WebSiteLanguage) Run(c *audit.Context) []audit.Issue {
	s := sitewideOf(c)
	if s == nil || s.site == nil || s.lang == "" {
		return nil
	}
	if _, ok := s.site.N.Get("inLanguage"); ok {
		return nil
	}
	return []audit.Issue{audit.Issuef(s.site.Path, audit.Warning, "this WebSite has no inLanguage: set it %s, the page's language", s.lang)}
}
