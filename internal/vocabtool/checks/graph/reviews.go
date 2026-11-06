package graph

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// ownKey is the memo key of the page's own ids.
type ownKey struct{}

// ownIDs are the @ids, and the plain strings, that the draft's publisher and
// provider properties name: the page's own business or person. The map is
// shared: read it only.
func ownIDs(c *audit.Context) map[string]bool {
	return audit.Memo(c, ownKey{}, func() map[string]bool {
		own := map[string]bool{}
		for _, l := range c.Nodes {
			for _, k := range []string{"publisher", "provider"} {
				raw, _ := l.N.Get(k)
				for _, v := range jsonld.Values(raw) {
					switch x := v.(type) {
					case *jsonld.Object:
						for _, id := range jsonld.Strs(x, "@id") {
							own[id] = true
						}
					case string:
						own[x] = true
					}
				}
			}
		}
		return own
	})
}

// reviewsKey is the memo key of the described Reviews.
type reviewsKey struct{}

// reviewsOf are the Review nodes that say more than their @id and type, in
// the order the draft lists them. A bare reference to a Review is checked
// where the Review itself is. The list is shared: read it only.
func reviewsOf(c *audit.Context) []jsonld.Node {
	return audit.Memo(c, reviewsKey{}, func() []jsonld.Node {
		var out []jsonld.Node
		for _, l := range c.Nodes {
			if c.Is(l.N, "Review") && len(l.N.Keys) >= 2 {
				out = append(out, l)
			}
		}
		return out
	})
}
