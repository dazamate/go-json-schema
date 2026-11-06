package shop

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(SortOrder{}) }

var (
	sortAsc  = regexp.MustCompile(`(?i)low to high|ascending|a\s*-\s*z|oldest`)
	sortDesc = regexp.MustCompile(`(?i)high to low|descending|z\s*-\s*a|newest`)
)

// SortOrder warns when a page that lists products has a sort control showing
// an ascending or descending order and no ItemList has an itemListOrder.
type SortOrder struct{}

// ID is the check's ID.
func (SortOrder) ID() string { return "shop/sort-order" }

// Run reports the first sort control whose chosen option names an order.
func (SortOrder) Run(c *audit.Context) []audit.Issue {
	if listed(c) == nil || c.Page.Doc == nil || jsonld.HasKey(c.Nodes, "itemListOrder") {
		return nil
	}
	doc := c.Page.Doc
	for _, sel := range webpage.QueryAll(doc, "select") {
		name, _ := webpage.Attr(sel, "name")
		id, _ := webpage.Attr(sel, "id")
		if !strings.Contains(strings.ToLower(name+" "+id), "sort") {
			continue
		}
		for _, o := range webpage.QueryAll(sel, "option[selected]") {
			t := webpage.HTMLText(o)
			member := ""
			switch {
			case sortAsc.MatchString(t):
				member = "ItemListOrderAscending"
			case sortDesc.MatchString(t):
				member = "ItemListOrderDescending"
			}
			if member != "" {
				return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(doc, sel), audit.Warning, "the page sorts its products %q: set the ItemList's itemListOrder https://schema.org/%s (an ItemListOrderType)", t, member)}
			}
		}
	}
	return nil
}
