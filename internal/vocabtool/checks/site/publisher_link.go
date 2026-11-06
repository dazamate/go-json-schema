package site

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(PublisherLink{}) }

// PublisherLink warns when no node publishes the page but the draft has the
// Organization that could: nothing ties the site to its business until the
// WebSite names it as its publisher. It needs the page's document, on a page
// search engines index.
type PublisherLink struct{}

// ID is the check's ID.
func (PublisherLink) ID() string { return "site/publisher-link" }

// Run asks for the draft's Organization as the WebSite's publisher.
func (PublisherLink) Run(c *audit.Context) []audit.Issue {
	g := gap(c)
	if g == nil || g.org == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "no node publishes the page: set the Organization %q as the WebSite's publisher (and the copyrightHolder), by its @id", g.org)}
}
