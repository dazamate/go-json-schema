package terms

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(UntypedNode{}) }

// UntypedNode warns on a node with properties but no @type: nothing says
// what it describes, so search engines cannot read its properties. A node of
// @ keywords alone, such as an {"@id"} reference, has none; nor does a node
// whose @id the graph types elsewhere (JSON-LD merges the two), or a
// breadcrumb's item, which Google reads as the crumb's URL and name.
type UntypedNode struct{}

// ID is the check's ID.
func (UntypedNode) ID() string { return "terms/untyped-node" }

// Run reports each untyped node with properties, inner nodes first.
func (UntypedNode) Run(c *audit.Context) []audit.Issue {
	typedIDs := map[string]bool{}
	crumbs := map[*jsonld.Object]bool{}
	for _, l := range c.Nodes {
		if _, typed := l.N.Get("@type"); typed {
			for _, id := range jsonld.Strs(l.N, "@id") {
				typedIDs[id] = true
			}
		}
		if !c.Is(l.N, "BreadcrumbList") {
			continue
		}
		raw, _ := l.N.Get("itemListElement")
		for _, v := range jsonld.Values(raw) {
			if li, ok := v.(*jsonld.Object); ok {
				item, _ := li.Get("item")
				for _, x := range jsonld.Values(item) {
					if o, ok := x.(*jsonld.Object); ok {
						crumbs[o] = true
					}
				}
			}
		}
	}
	var out []audit.Issue
	for _, n := range walkOf(c).nodes {
		if crumbs[n.n] || slices.ContainsFunc(jsonld.Strs(n.n, "@id"), func(id string) bool { return typedIDs[id] }) {
			continue
		}
		props := slices.ContainsFunc(n.n.Keys, func(k string) bool { return !strings.HasPrefix(k, "@") })
		if _, typed := n.n.Get("@type"); !typed && props {
			out = append(out, audit.Issuef(n.path, audit.Warning, "node has properties but no @type"))
		}
	}
	return out
}
