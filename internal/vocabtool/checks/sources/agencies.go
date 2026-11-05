package sources

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Agencies{}) }

// Agencies warns on the government agencies the page links that no value in
// the draft names: their sites, by host. Each is a GovernmentOrganization. It
// needs the page's document, on a page search engines index.
type Agencies struct{}

// ID is the check's ID.
func (Agencies) ID() string { return "sources/agencies" }

// Run lists the agencies no value names, in one issue.
func (Agencies) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	d := c.Index()
	hosts := map[string]bool{}
	for k := range d.URLs {
		if u, err := url.Parse(k); err == nil {
			hosts[strings.TrimPrefix(u.Hostname(), "www.")] = true
		}
	}
	var list []string
	seen := map[string]bool{}
	for _, a := range webpage.QueryAll(c.Page.Doc, "a[href]") {
		u, ok := govLink(a)
		host := ""
		if ok {
			host = strings.TrimPrefix(u.Hostname(), "www.")
		}
		if !ok || seen[host] || hosts[host] || ownSite(c, host) {
			continue
		}
		seen[host] = true
		// a link names the agency, or else the page it links to
		name := webpage.HTMLText(a)
		if d.HoldsText(name) {
			continue
		}
		if n := len(strings.Fields(name)); n == 0 || n > 8 || !agencyName.MatchString(name) {
			name = host
		}
		list = append(list, fmt.Sprintf("%q (%s)", name, u.Hostname()))
	}
	if len(list) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page links the agencies %s: add each as a GovernmentOrganization (name, url) in the WebPage's mentions, or as the publisher of a citation from it", txt.Examples(list))}
}

// ownSite reports a host that is the page's own site or one of its
// subdomains: a government site's portals are no other agency.
func ownSite(c *audit.Context, host string) bool {
	if c.Page.Base == nil {
		return false
	}
	own := strings.TrimPrefix(c.Page.Base.Hostname(), "www.")
	return host == own || strings.HasSuffix(host, "."+own)
}
