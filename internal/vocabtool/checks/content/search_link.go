package content

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(SearchLink{}) }

// SearchLink warns on a link to the site's search page when the page has no
// search box to ask for a SearchAction and the draft has none. It needs the
// page's document, on a page search engines index.
type SearchLink struct{}

// ID is the check's ID.
func (SearchLink) ID() string { return "content/search-link" }

// Run reports the first link to the search page.
func (SearchLink) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	s := searchOf(c)
	if s.link == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, s.link), audit.Warning, "the page links the site's search page (%s) but no SearchAction: add one to the WebSite's potentialAction, target an EntryPoint whose urlTemplate is the search page's address with {search_term_string} as the query", s.to)}
}
