package site

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(PublisherAdd{}) }

// PublisherAdd warns when no node publishes the page and the draft has no
// Organization to do it: a site is published by a business or a person, and
// the markup that names neither leaves search engines to guess. The ask names
// the © line's holder when the page has one. It needs the page's document, on
// a page search engines index.
type PublisherAdd struct{}

// ID is the check's ID.
func (PublisherAdd) ID() string { return "site/publisher-add" }

// Run asks for the business as an Organization, and as the WebSite's publisher.
func (PublisherAdd) Run(c *audit.Context) []audit.Issue {
	g := gap(c)
	if g == nil || g.org != "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "no node publishes the page: add the business as an Organization (or the subtype that fits it, or the Person who runs the site) with %s, url, logo, and the page's telephone and email, and set it as the WebSite's publisher", g.name)}
}
