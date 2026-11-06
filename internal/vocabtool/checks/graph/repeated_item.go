package graph

import (
	"fmt"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(RepeatedItem{}) }

// RepeatedItem warns about a node without an @id that names the same Wikidata
// item as a node with one: it repeats that node, which a {"@id"} reference
// says once. Only a node that says more than its @id counts as the original.
type RepeatedItem struct{}

// ID is the check's ID.
func (RepeatedItem) ID() string { return "graph/repeated-item" }

// Run reports each repeat of a node with an @id.
func (RepeatedItem) Run(c *audit.Context) []audit.Issue {
	byItem := map[string]string{}
	var collect func(v any)
	collect = func(v any) {
		switch x := v.(type) {
		case []any:
			for _, item := range x {
				collect(item)
			}
		case *jsonld.Object:
			if id, ok := x.Get("@id"); ok && len(x.Keys) > 1 {
				if s, ok := id.(string); ok {
					for _, q := range jsonld.WikidataItems(x) {
						if _, seen := byItem[q]; !seen {
							byItem[q] = s
						}
					}
				}
			}
			for _, k := range x.Keys {
				collect(x.Vals[k])
			}
		}
	}
	collect(c.Root)
	if len(byItem) == 0 {
		return nil
	}
	var out []audit.Issue
	var check func(path string, v any)
	check = func(path string, v any) {
		switch x := v.(type) {
		case []any:
			for i, item := range x {
				check(fmt.Sprintf("%s[%d]", path, i), item)
			}
		case *jsonld.Object:
			if _, ok := x.Get("@id"); !ok {
				for _, q := range jsonld.WikidataItems(x) {
					if id, ok := byItem[q]; ok {
						out = append(out, audit.Issuef(path, audit.Warning, "same Wikidata item (%s) as the node %s: put {\"@id\": %q} here instead of repeating it", q, id, id))
						break
					}
				}
			}
			for _, k := range x.Keys {
				if k != "@context" {
					check(path+"."+k, x.Vals[k])
				}
			}
		}
	}
	check("$", c.Root)
	return out
}
