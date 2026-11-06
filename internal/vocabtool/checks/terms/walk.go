// Package terms holds the vocabulary checks: unknown and superseded types and
// properties, keys that are not schema.org terms, enumeration members,
// numbers written as text and properties used outside their types. Check and
// Vocabulary both run them. They share one walk of the draft, which skips
// what the vocabulary cannot check: it goes no further into a Google
// annotation, a key that is not a schema.org term or an unknown property.
package terms

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/schemagen"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

// walked is what the vocabulary walk visits, each list in the order it
// visits them. The walk starts at the draft's root, a node's @type names
// before its keys, and a key's values after it. The lists are shared: read
// them only.
type walked struct {
	types       []typeName // every @type name of every node
	annotations []key      // Google's action annotations: the walk goes no further
	foreign     []key      // keys that are not schema.org terms: no further
	unknown     []key      // keys that name no property: no further
	props       []key      // the properties the vocabulary has: their values are walked
	values      []value    // every text value of those properties
	nodes       []node     // every node, once its keys are walked
}

// typeName is one name in a node's @type.
type typeName struct {
	path  string           // the node's path
	name  string           // the name as written
	term  string           // the name without a schema.org prefix
	class *schemagen.Class // the type, nil when the vocabulary lacks it
}

// key is one key of a node, other than an @ keyword.
type key struct {
	path  string          // the key's path
	name  string          // the key as written
	term  string          // the key without a schema.org prefix
	node  *jsonld.Object  // the node that holds it
	types []string        // the node's types the vocabulary has
	prop  *schemagen.Prop // the property, in props only
}

// value is one text value of a known property; a value that is a node or a
// list is walked instead.
type value struct {
	path string // the value's path
	prop string // the property's term
	text string // the value
}

// node is a node the walk visits.
type node struct {
	path string
	n    *jsonld.Object
}

// walkKey is the memo key of the walk.
type walkKey struct{}

// walkOf is the walk of the context's draft, made once. It starts at the root
// as given, so an array root that Vocabulary checks keeps its $[i] paths.
func walkOf(c *audit.Context) *walked {
	return audit.Memo(c, walkKey{}, func() *walked {
		w := &walked{}
		w.walk(c.Vocab, "$", c.Root)
		return w
	})
}

// walk visits a value: each item of a list, and an object's @type names, then
// its keys, @graph included.
func (w *walked) walk(v *vocab.Vocab, path string, x any) {
	switch x := x.(type) {
	case []any:
		for i, item := range x {
			w.walk(v, fmt.Sprintf("%s[%d]", path, i), item)
		}
	case *jsonld.Object:
		types := w.typesOf(v, path, x)
		for _, k := range x.Keys {
			if strings.HasPrefix(k, "@") {
				if k == "@graph" {
					w.walk(v, path+".@graph", x.Vals[k])
				}
				continue
			}
			w.key(v, key{path: path + "." + k, name: k, term: jsonld.StripPrefix(k), node: x, types: types})
		}
		w.nodes = append(w.nodes, node{path, x})
	}
}

// typesOf notes the node's @type names and returns those the vocabulary has.
func (w *walked) typesOf(v *vocab.Vocab, path string, n *jsonld.Object) []string {
	raw, ok := n.Get("@type")
	if !ok {
		return nil
	}
	var names []string
	switch x := raw.(type) {
	case string:
		names = []string{x}
	case []any:
		for _, e := range x {
			if s, ok := e.(string); ok {
				names = append(names, s)
			}
		}
	}
	var known []string
	for _, name := range names {
		t := typeName{path: path, name: name, term: jsonld.StripPrefix(name), class: v.V.Classes[jsonld.StripPrefix(name)]}
		w.types = append(w.types, t)
		if t.class != nil {
			known = append(known, t.term)
		}
	}
	return known
}

// key notes a node's key in the list for what it is, then walks a known
// property's values.
func (w *walked) key(v *vocab.Vocab, k key) {
	switch {
	case audit.GoogleAnnotation.MatchString(k.term):
		w.annotations = append(w.annotations, k)
		return
	case strings.Contains(k.term, ":") || strings.Contains(k.term, "/"):
		w.foreign = append(w.foreign, k)
		return
	}
	p, ok := v.V.Props[k.term]
	if !ok {
		w.unknown = append(w.unknown, k)
		return
	}
	k.prop = p
	w.props = append(w.props, k)
	switch x := k.node.Vals[k.name].(type) {
	case []any:
		for i, item := range x {
			w.value(v, fmt.Sprintf("%s[%d]", k.path, i), k.term, item)
		}
	default:
		w.value(v, k.path, k.term, x)
	}
}

// value notes a property's text value, and walks any other.
func (w *walked) value(v *vocab.Vocab, path, prop string, x any) {
	s, ok := x.(string)
	if !ok {
		w.walk(v, path, x)
		return
	}
	w.values = append(w.values, value{path, prop, s})
}

// unlisted reports a property that none of its node's known types takes,
// when the node has one: a subtype that takes it may be what the node means.
func (k key) unlisted(v *vocab.Vocab) bool {
	return len(k.types) > 0 && !slices.ContainsFunc(k.types, func(ty string) bool { return v.Applicable(ty, k.term) })
}
