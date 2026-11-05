package snippets

import (
	"fmt"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// product is a Product node the draft describes, the one the checks hold to
// the snippet requirements.
type product struct {
	jsonld.Node
	ty string // its first type, as written
}

// offer is one of a Product's offers, once a reference is followed.
type offer struct {
	path  string         // where the Product holds it: its offers, or an item of them
	n     *jsonld.Object // the offer
	specs []spec         // the price specifications it nests
}

// spec is a price specification nested in an offer.
type spec struct {
	path string
	n    *jsonld.Object
}

// productsKey, resolverKey and offersKey are the memo keys.
type (
	productsKey struct{}
	resolverKey struct{}
	offersKey   struct{}
)

// productsOf is every described node that is a Product or a subtype of it, a
// ProductGroup included, in the order the draft lists them. The list is
// shared: read it only.
func productsOf(c *audit.Context) []product {
	return audit.Memo(c, productsKey{}, func() []product {
		var out []product
		for _, l := range c.Nodes {
			if jsonld.Described(l.N) && c.Is(l.N, "Product") {
				out = append(out, product{l, jsonld.TypeNames(l.N)[0]})
			}
		}
		return out
	})
}

// resolver is the node a value is, following a bare {"@id"} reference to the
// described node with that @id.
func resolver(c *audit.Context) func(any) *jsonld.Object {
	byID := audit.Memo(c, resolverKey{}, func() map[string]*jsonld.Object {
		byID := map[string]*jsonld.Object{}
		for _, l := range c.Nodes {
			if jsonld.Described(l.N) {
				for _, id := range jsonld.Strs(l.N, "@id") {
					if _, seen := byID[id]; !seen {
						byID[id] = l.N
					}
				}
			}
		}
		return byID
	})
	return func(v any) *jsonld.Object {
		o, ok := v.(*jsonld.Object)
		if !ok {
			return nil
		}
		if !jsonld.Described(o) {
			for _, id := range jsonld.Strs(o, "@id") {
				if n, ok := byID[id]; ok {
					return n
				}
			}
		}
		return o
	}
}

// offersOf is every offer of every product, with the price specifications it
// nests, in the order the draft lists the products. An offer two products
// share is listed under each. The list is shared: read it only.
func offersOf(c *audit.Context) []offer {
	return audit.Memo(c, offersKey{}, func() []offer {
		resolve := resolver(c)
		var out []offer
		for _, p := range productsOf(c) {
			raw, _ := p.N.Get("offers")
			for i, v := range jsonld.Values(raw) {
				path := p.Path + ".offers"
				if _, list := raw.([]any); list {
					path = fmt.Sprintf("%s[%d]", path, i)
				}
				if o := resolve(v); o != nil {
					out = append(out, offer{path, o, specsOf(path, o, resolve)})
				}
			}
		}
		return out
	})
}

// specsOf are the price specifications an offer nests.
func specsOf(path string, o *jsonld.Object, resolve func(any) *jsonld.Object) []spec {
	var out []spec
	raw, _ := o.Get("priceSpecification")
	for i, v := range jsonld.Values(raw) {
		if s := resolve(v); s != nil {
			p := path + ".priceSpecification"
			if _, list := raw.([]any); list {
				p = fmt.Sprintf("%s[%d]", p, i)
			}
			out = append(out, spec{p, s})
		}
	}
	return out
}

// has reports whether the node sets the property.
func has(n *jsonld.Object, key string) bool {
	_, ok := n.Get(key)
	return ok
}

// specHas reports whether one of the offer's price specifications sets the
// property.
func (o offer) specHas(key string) bool {
	for _, s := range o.specs {
		if has(s.n, key) {
			return true
		}
	}
	return false
}
