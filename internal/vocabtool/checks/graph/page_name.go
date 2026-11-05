package graph

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(PageName{}) }

// PageName warns about a WebPage node, not a reference, that has no url or no
// name. It needs the page's facts but not an indexed page: the markup's own
// faults are held to a page search engines skip too.
type PageName struct{}

// ID is the check's ID.
func (PageName) ID() string { return "graph/page-name" }

// Run reports each WebPage missing its url or its name.
func (PageName) Run(c *audit.Context) []audit.Issue {
	if c.Page == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		if !c.Is(l.N, "WebPage") {
			continue
		}
		var miss []string
		for _, k := range []string{"url", "name"} {
			if _, ok := l.N.Get(k); !ok {
				miss = append(miss, k)
			}
		}
		if len(miss) > 0 {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "this %s has no %s: give it the page's", jsonld.TypeNames(l.N)[0], strings.Join(miss, " or ")))
		}
	}
	return out
}
