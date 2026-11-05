package keyword

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(LooseNodes{}) }

// looseProps are the properties whose targets are parts of another node: its
// entities, sources and sections. A top-level node only these point to reads
// as loose. The page's mainEntity and breadcrumb stand at the top on their
// own: a Product or an Article is an item a search engine looks for there.
var looseProps = []string{"about", "mentions", "knowsAbout", "areaServed", "spatialCoverage", "hasPart", "citation", "isBasedOn"}

// LooseNodes warns once on the top-level nodes that a node's entities,
// sources or sections point to, other than the page, the WebSite and the
// site's business. A flat @graph is valid JSON-LD, but validator.schema.org
// and Google's Rich Results Test list every top-level node as an item of its
// own, so the page's entities read as loose things with no parent. Nesting
// each where it is first referenced, keeping its @id, makes the graph one
// tree. It reads no page.
type LooseNodes struct{}

// ID is the check's ID.
func (LooseNodes) ID() string { return "keyword/loose-nodes" }

// Run reports the loose nodes, named, in one issue on the draft.
func (LooseNodes) Run(c *audit.Context) []audit.Issue {
	page, _ := audit.MainWebPage(c.Vocab, c.Nodes, c.Known)
	own := map[*jsonld.Object]bool{page.N: true}
	for _, l := range jsonld.Publishers(c.Nodes) {
		own[l.N] = true
	}
	referenced := map[string]bool{}
	for _, l := range c.Nodes {
		for _, k := range looseProps {
			raw, ok := l.N.Get(k)
			if !ok {
				continue
			}
			for _, v := range jsonld.Values(raw) {
				if o, ok := v.(*jsonld.Object); ok && len(o.Keys) == 1 {
					if id, ok := o.Get("@id"); ok {
						s, _ := id.(string)
						referenced[s] = true
					}
				}
			}
		}
	}
	var loose []string
	for _, l := range c.Nodes {
		if !jsonld.TopLevel(l.Path) || own[l.N] || slices.Contains(jsonld.TypeNames(l.N), "WebSite") || len(l.N.Keys) < 2 {
			continue
		}
		ids := jsonld.Strs(l.N, "@id")
		if len(ids) == 0 || !referenced[ids[0]] {
			continue
		}
		name := strings.Join(append(jsonld.Strs(l.N, "name"), jsonld.Strs(l.N, "headline")...), " ")
		if name == "" {
			name = ids[0]
		}
		loose = append(loose, name)
	}
	if len(loose) == 0 {
		return nil
	}
	shown := loose
	if len(shown) > 8 {
		shown = append(slices.Clone(shown[:8]), fmt.Sprintf("and %d more", len(loose)-8))
	}
	return []audit.Issue{audit.Issuef("$", audit.Warning, "%d top-level nodes are referenced from other nodes (%s): validator.schema.org lists each top-level node as an item of its own, so they read as loose things with no parent. Nest each where it is first referenced, keeping its @id, and leave only the page, the WebSite and the business at the top", len(loose), strings.Join(shown, ", "))}
}
