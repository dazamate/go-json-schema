package graph

import (
	"net/url"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// ownKey is the memo key of the page's own ids.
type ownKey struct{}

// ownSet is the page's own business or person: the @ids and plain strings
// that name it, and its nodes.
type ownSet struct {
	ids   map[string]bool
	nodes map[*jsonld.Object]bool
}

// has reports whether a value (a node or an @id) is the page's own.
func (o *ownSet) has(v any) bool {
	switch x := v.(type) {
	case *jsonld.Object:
		return o.nodes[x] || slices.ContainsFunc(jsonld.Strs(x, "@id"), func(id string) bool { return o.ids[id] })
	case string:
		return o.ids[x]
	}
	return false
}

// add makes a value (a node or an @id) the page's own.
func (o *ownSet) add(v any) {
	switch x := v.(type) {
	case *jsonld.Object:
		o.nodes[x] = true
		for _, id := range jsonld.Strs(x, "@id") {
			o.ids[id] = true
		}
	case string:
		o.ids[x] = true
	}
}

// ownOf is the page's own business or person: what the draft's publisher
// and provider properties name; an Organization whose url is the site's
// home page, or that is named as the WebSite; and the departments and
// sub-organizations of any of them. The set is shared: read it only.
func ownOf(c *audit.Context) *ownSet {
	return audit.Memo(c, ownKey{}, func() *ownSet {
		own := &ownSet{ids: map[string]bool{}, nodes: map[*jsonld.Object]bool{}}
		var sites []string
		for _, l := range c.Nodes {
			for _, k := range []string{"publisher", "provider"} {
				raw, _ := l.N.Get(k)
				for _, v := range jsonld.Values(raw) {
					own.add(v)
				}
			}
			if c.Is(l.N, "WebSite") {
				sites = append(sites, jsonld.Strs(l.N, "name")...)
			}
		}
		for _, l := range c.Nodes {
			if !c.Is(l.N, "Organization") {
				continue
			}
			if slices.ContainsFunc(jsonld.Strs(l.N, "url"), func(u string) bool { return siteHome(c, u) }) ||
				slices.ContainsFunc(jsonld.Strs(l.N, "name"), func(n string) bool { return slices.Contains(sites, n) }) {
				own.add(l.N)
			}
		}
		for _, l := range c.Nodes {
			if !own.has(l.N) {
				continue
			}
			for _, k := range []string{"department", "subOrganization"} {
				raw, _ := l.N.Get(k)
				for _, v := range jsonld.Values(raw) {
					own.add(v)
				}
			}
		}
		return own
	})
}

// siteHome reports an address that is the home page of the page's site.
func siteHome(c *audit.Context, s string) bool {
	if c.Page == nil || c.Page.Base == nil {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(s))
	return err == nil && strings.EqualFold(u.Hostname(), c.Page.Base.Hostname()) && strings.Trim(u.Path, "/") == ""
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
