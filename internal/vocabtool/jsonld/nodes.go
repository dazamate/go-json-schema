package jsonld

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/classtype"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// Node is a node and its path in the draft.
type Node struct {
	Path string
	N    *Object
}

// Nodes lists every object in the draft with its path, outermost first.
func Nodes(path string, v any) []Node {
	var out []Node
	switch x := v.(type) {
	case []any:
		for i, item := range x {
			out = append(out, Nodes(fmt.Sprintf("%s[%d]", path, i), item)...)
		}
	case *Object:
		out = append(out, Node{path, x})
		for _, k := range x.Keys {
			if k != "@context" {
				out = append(out, Nodes(path+"."+k, x.Vals[k])...)
			}
		}
	}
	return out
}

// TypeNames are the node's @type values without a schema: prefix.
func TypeNames(n *Object) []string {
	raw, _ := n.Get("@type")
	var out []string
	for _, v := range Values(raw) {
		if s, ok := v.(string); ok {
			out = append(out, StripPrefix(s))
		}
	}
	return out
}

// Values is a property's value as a list: a single value is a list of one.
func Values(raw any) []any {
	switch x := raw.(type) {
	case nil:
		return nil
	case []any:
		return x
	default:
		return []any{x}
	}
}

// Strs are a property's string values.
func Strs(n *Object, key string) []string {
	raw, _ := n.Get(key)
	var out []string
	for _, v := range Values(raw) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// Scalars are a property's string, number and boolean values, as text.
func Scalars(n *Object, key string) []string {
	raw, _ := n.Get(key)
	var out []string
	for _, v := range Values(raw) {
		switch x := v.(type) {
		case string:
			out = append(out, x)
		case fmt.Stringer: // json.Number
			out = append(out, x.String())
		case bool:
			out = append(out, strconv.FormatBool(x))
		}
	}
	return out
}

// Index holds what the content checks look up in the draft.
type Index struct {
	Strs []string           // every string value
	URLs map[string]bool    // every string value, as a normalized URL
	Path map[*Object]string // every node's path
}

// NewIndex indexes the draft's nodes.
func NewIndex(all []Node) *Index {
	d := &Index{URLs: map[string]bool{}, Path: map[*Object]string{}}
	for _, l := range all {
		d.Path[l.N] = l.Path
		for _, k := range l.N.Keys {
			if strings.HasPrefix(k, "@") {
				continue
			}
			for _, v := range Values(l.N.Vals[k]) {
				if s, ok := v.(string); ok {
					d.Strs = append(d.Strs, s)
					d.URLs[txt.NormLink(s)] = true
				}
			}
		}
	}
	return d
}

// HoldsText reports a draft string that holds s, words compared.
func (d *Index) HoldsText(s string) bool {
	w := txt.NormWords(s)
	return w != "" && slices.ContainsFunc(d.Strs, func(v string) bool { return strings.Contains(" "+txt.NormWords(v)+" ", " "+w+" ") })
}

// HasKey reports a node with the property in the draft.
func HasKey(all []Node, key string) bool {
	return slices.ContainsFunc(all, func(l Node) bool { _, ok := l.N.Get(key); return ok })
}

// KeyMember reports a node whose property names the enumeration member, by
// its URL or its bare name.
func KeyMember(all []Node, key string, members ...string) bool {
	return slices.ContainsFunc(all, func(l Node) bool {
		return slices.ContainsFunc(Strs(l.N, key), func(s string) bool { return slices.Contains(members, StripPrefix(s)) })
	})
}

// DeepScalars are every string, number and boolean under v, as text.
func DeepScalars(v any) []string {
	var out []string
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			out = append(out, DeepScalars(e)...)
		}
	case *Object:
		for _, k := range x.Keys {
			if !strings.HasPrefix(k, "@") {
				out = append(out, DeepScalars(x.Vals[k])...)
			}
		}
	case string:
		out = append(out, x)
	case fmt.Stringer:
		out = append(out, x.String())
	case bool:
		out = append(out, strconv.FormatBool(x))
	}
	return out
}

// Under are the scalars under the property on every node that has it.
func Under(all []Node, key string) []string {
	var out []string
	for _, l := range all {
		if v, ok := l.N.Get(key); ok {
			out = append(out, DeepScalars(v)...)
		}
	}
	return out
}

// ByID finds a node in the draft by its @id.
func ByID(all []Node, id string) (Node, bool) {
	for _, l := range all {
		if slices.Contains(Strs(l.N, "@id"), id) && len(l.N.Keys) > 1 {
			return l, true
		}
	}
	return Node{}, false
}

// Refs are the nodes a property points at: nested nodes, and the nodes an
// {"@id"} reference names.
func Refs(all []Node, n *Object, key string) []Node {
	raw, _ := n.Get(key)
	var out []Node
	for _, v := range Values(raw) {
		o, ok := v.(*Object)
		if !ok {
			continue
		}
		if len(o.Keys) > 1 {
			if i := slices.IndexFunc(all, func(l Node) bool { return l.N == o }); i >= 0 {
				out = append(out, all[i])
			}
			continue
		}
		for _, id := range Strs(o, "@id") {
			if l, ok := ByID(all, id); ok {
				out = append(out, l)
			}
		}
	}
	return out
}

// Publishers are the nodes the draft names as the site's publisher or
// copyright holder.
func Publishers(all []Node) []Node {
	var out []Node
	for _, l := range all {
		for _, k := range []string{"publisher", "copyrightHolder"} {
			for _, r := range Refs(all, l.N, k) {
				if !slices.ContainsFunc(out, func(o Node) bool { return o.N == r.N }) {
					out = append(out, r)
				}
			}
		}
	}
	return out
}

// Described reports a node that says something itself, not only a reference
// to a node elsewhere: a property beyond @id and @type.
func Described(n *Object) bool {
	for _, k := range n.Keys {
		if !strings.HasPrefix(k, "@") {
			return true
		}
	}
	return false
}

// OwnItems are the Wikidata items a node is: its sameAs links and its
// identifiers whose propertyID is Wikidata.
func OwnItems(n *Object) []string {
	var out []string
	for _, s := range Strs(n, "sameAs") {
		if m := WikidataItem.FindStringSubmatch(s); m != nil {
			out = append(out, m[1])
		}
	}
	raw, _ := n.Get("identifier")
	for _, v := range Values(raw) {
		if pv, ok := v.(*Object); ok && slices.ContainsFunc(Strs(pv, "propertyID"), func(s string) bool { return strings.EqualFold(s, "Wikidata") }) {
			out = append(out, Strs(pv, "value")...)
		}
	}
	return out
}

// HasWikidataClass reports a Wikidata class in the node's additionalType that
// the classtype table maps: classTypes types such a node, not its name (the
// Blue Mountains are a region, not a Mountain). A topic class the table does
// not map, such as wedding on a WebPage, leaves the name signal on.
func HasWikidataClass(n *Object) bool {
	return slices.ContainsFunc(Strs(n, "additionalType"), func(cl string) bool {
		m := WikidataItem.FindStringSubmatch(cl)
		return m != nil && classtype.Of([]string{m[1]}, false) != ""
	})
}
