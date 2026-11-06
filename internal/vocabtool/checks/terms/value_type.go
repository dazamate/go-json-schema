package terms

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(ValueType{}) }

// ValueType errs on a node given to a property that takes none of its types,
// such as an ItemList in hasPart, which takes a CreativeWork: the Schema
// Markup Validator reports it as an invalid object. A reference is read as
// the node of its @id the draft defines. A Role stands in for any value, and
// a property that takes only text, numbers and dates is left to the literal
// checks.
type ValueType struct{}

// ID is the check's ID.
func (ValueType) ID() string { return "terms/value-type" }

// Run reports each node value whose known types all fall outside the types
// its property takes.
func (ValueType) Run(c *audit.Context) []audit.Issue {
	byID := map[string][]string{}
	for _, l := range c.Nodes {
		id, _ := l.N.Get("@id")
		if s, ok := id.(string); ok && s != "" && len(byID[s]) == 0 {
			byID[s] = known(c, jsonld.TypeNames(l.N))
		}
	}
	var out []audit.Issue
	for _, k := range walkOf(c).props {
		takes := nodeRanges(c, k.prop.Ranges)
		if len(takes) == 0 || slices.Contains(takes, "Thing") {
			continue
		}
		raw := k.node.Vals[k.name]
		for i, x := range jsonld.Values(raw) {
			n, ok := x.(*jsonld.Object)
			if !ok {
				continue
			}
			types := known(c, jsonld.TypeNames(n))
			if _, typed := n.Get("@type"); !typed {
				if id, ok := n.Get("@id"); ok {
					s, _ := id.(string)
					types = byID[s]
				}
			}
			if len(types) == 0 || c.Vocab.Is(types, "Role") || slices.ContainsFunc(takes, func(t string) bool { return c.Vocab.Is(types, t) }) {
				continue
			}
			path := k.path
			if _, list := raw.([]any); list {
				path = fmt.Sprintf("%s[%d]", k.path, i)
			}
			got := strings.Join(types, ", ")
			out = append(out, audit.Issuef(path, audit.Error, "%s takes %s, not %s: give it a node of a type it takes, or move the %s to a property that takes it", k.term, strings.Join(takes, " or "), got, got))
		}
	}
	return out
}

// known are the types the vocabulary has.
func known(c *audit.Context, types []string) []string {
	return slices.DeleteFunc(slices.Clone(types), func(t string) bool { return c.Vocab.V.Classes[t] == nil })
}

// nodeRanges are the types a property takes that a node can be, its data
// types (Text, URL, Number, Date…) left out.
func nodeRanges(c *audit.Context, ranges []string) []string {
	var out []string
	for _, r := range ranges {
		if c.Vocab.V.Classes[r] == nil {
			continue
		}
		// URL and Integer are data types through Text and Number
		literal := slices.ContainsFunc(c.Vocab.V.Ancestors(r), func(a string) bool {
			cl := c.Vocab.V.Classes[a]
			return a == "DataType" || (cl != nil && cl.DataType)
		})
		if !literal {
			out = append(out, r)
		}
	}
	return out
}
