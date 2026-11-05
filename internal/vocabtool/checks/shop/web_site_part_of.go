package shop

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(WebSitePartOf{}) }

// WebSitePartOf warns when the page's WebPage has no isPartOf, and the draft
// has the WebSite it should name: the WebPage belongs to the site.
type WebSitePartOf struct{}

// ID is the check's ID.
func (WebSitePartOf) ID() string { return "shop/web-site-part-of" }

// Run reports the WebPage, naming the WebSite's @id (or its url).
func (WebSitePartOf) Run(c *audit.Context) []audit.Issue {
	s := sitewideOf(c)
	if s == nil || s.site == nil || s.page == nil {
		return nil
	}
	if _, ok := s.page.N.Get("isPartOf"); ok {
		return nil
	}
	ref := strings.Join(append(jsonld.Strs(s.site.N, "@id"), jsonld.Strs(s.site.N, "url")...), ", ")
	return []audit.Issue{audit.Issuef(s.page.Path, audit.Warning, "the WebPage has no isPartOf: set it to the WebSite ({\"@id\": %q})", txt.OrDefault(strings.SplitN(ref, ", ", 2)[0], "the WebSite's @id"))}
}
