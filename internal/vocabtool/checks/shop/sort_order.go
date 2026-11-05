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
	sortAsc  = regexp.MustCompile(`(?i)low(?:est)?\s*(?:to|-|–)\s*high(?:est)?|(?:lowest|cheapest)(?: price)? first|ascending|a\s*-\s*z|oldest|old to new`)
	sortDesc = regexp.MustCompile(`(?i)high(?:est)?\s*(?:to|-|–)\s*low(?:est)?|highest(?: price)? first|descending|z\s*-\s*a|newest|new to old`)
	// sortControl is what names a select that sorts: its name, id, class or
	// label says sort or order ("sort_by", "SortBy", "orderby"), not "border".
	sortControl = regexp.MustCompile(`(?i)sort|(?:^|[^a-z])order(?:by|ing)?(?:[^a-z]|$)`)
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
		var names []string
		for _, a := range []string{"name", "id", "class", "aria-label"} {
			v, _ := webpage.Attr(sel, a)
			names = append(names, v)
		}
		if !sortControl.MatchString(strings.Join(names, " ")) {
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
