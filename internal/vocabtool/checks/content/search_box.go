package content

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(SearchBox{}) }

// SearchBox warns on a search box the page shows when the draft has no
// SearchAction. The SearchAction is what lets a search engine offer the
// site's search in its results. It needs the page's document, on a page
// search engines index.
type SearchBox struct{}

// ID is the check's ID.
func (SearchBox) ID() string { return "content/search-box" }

// Run reports the first search box.
func (SearchBox) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	s := searchOf(c)
	if s.box == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, s.box), audit.Warning, "the page has a search box but no SearchAction: add one to the WebSite's potentialAction, with a target urlTemplate holding {search_term_string}")}
}
