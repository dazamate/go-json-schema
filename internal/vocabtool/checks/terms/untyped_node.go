package terms

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(UntypedNode{}) }

// UntypedNode warns on a node with properties but no @type: nothing says
// what it describes, so search engines cannot read its properties. A node of
// @ keywords alone, such as an {"@id"} reference, has none.
type UntypedNode struct{}

// ID is the check's ID.
func (UntypedNode) ID() string { return "terms/untyped-node" }

// Run reports each untyped node with properties, inner nodes first.
func (UntypedNode) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, n := range walkOf(c).nodes {
		props := slices.ContainsFunc(n.n.Keys, func(k string) bool { return !strings.HasPrefix(k, "@") })
		if _, typed := n.n.Get("@type"); !typed && props {
			out = append(out, audit.Issuef(n.path, audit.Warning, "node has properties but no @type"))
		}
	}
	return out
}
