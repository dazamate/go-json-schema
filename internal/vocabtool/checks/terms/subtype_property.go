package terms

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(SubtypeProperty{}) }

var _ audit.Retyper = SubtypeProperty{}

// SubtypeProperty warns on a property its node's type does not list but a
// subtype of the type does, such as openingHours on an Organization: the
// subtype is the type the node means. It is the first of the retype ledger's
// sources, and it tells the node; the later sources leave a told node be.
type SubtypeProperty struct{}

// ID is the check's ID.
func (SubtypeProperty) ID() string { return "terms/subtype-property" }

// subtyped is a property only subtypes of its node's types take, and the
// most general of those subtypes.
type subtyped struct {
	key
	subs []string
}

// find is every property the subtypes of its node's types take instead.
func (SubtypeProperty) find(c *audit.Context) []subtyped {
	var out []subtyped
	for _, k := range walkOf(c).props {
		if !k.unlisted(c.Vocab) {
			continue
		}
		if subs := c.Vocab.SubtypesTaking(k.types, k.term); len(subs) > 0 {
			out = append(out, subtyped{k, subs})
		}
	}
	return out
}

// Run reports each property with the subtypes that take it.
func (s SubtypeProperty) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range s.find(c) {
		noun := "subtype"
		if len(f.subs) > 1 {
			noun = "subtypes"
		}
		out = append(out, audit.Issuef(f.path, audit.Warning, "%s is not listed for %s but is for its %s %s: type it %s", f.term, strings.Join(f.types, ", "), noun, strings.Join(f.subs, " and "), strings.Join(f.subs, " or ")))
	}
	return out
}

// Retypes tells each node a property of it gives a subtype to, and names
// those subtypes.
func (s SubtypeProperty) Retypes(c *audit.Context, _ audit.Retyped) []audit.Retype {
	var out []audit.Retype
	for _, f := range s.find(c) {
		out = append(out, audit.Retype{Node: f.node, Named: f.subs})
	}
	return out
}
