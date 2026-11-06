package content

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Copyright{}) }

// Copyright warns when the page shows a copyright year the markup lacks, or
// has the year and no copyrightHolder. A search engine reads who owns the
// page's content from them. It runs on a page search engines index.
type Copyright struct{}

// ID is the check's ID.
func (Copyright) ID() string { return "content/copyright" }

// Run reports the page's WebPage when the copyright line is not in the markup.
func (Copyright) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	m := webpage.CopyrightLine.FindStringSubmatch(c.Page.Text)
	first, year, shown := "", "", ""
	switch {
	case m != nil:
		first, year, shown = m[1], m[2], m[0]
	default:
		// the year before the mark
		if m = webpage.CopyrightYearFirst.FindStringSubmatch(c.Page.Text); m == nil {
			return nil
		}
		year, shown = m[1], m[0]
	}
	hasYear, hasHolder := false, false
	for _, l := range c.Nodes {
		// a range shows the year the page began and the year it is now: the
		// markup names either
		for _, y := range jsonld.Scalars(l.N, "copyrightYear") {
			hasYear = hasYear || y == year || (first != "" && y == first)
		}
		if _, ok := l.N.Get("copyrightHolder"); ok {
			hasHolder = true
		}
	}
	if hasYear && hasHolder {
		return nil
	}
	return []audit.Issue{audit.Issuef(pageNode(c), audit.Warning, "the page shows %q but the markup has no copyrightYear %s and copyrightHolder: set both on the WebPage or the WebSite", strings.TrimSpace(shown), year)}
}
