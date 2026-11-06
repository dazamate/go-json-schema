package graph

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/classtype"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(ClassType{}) }

var _ audit.Retyper = ClassType{}

// ClassType warns about a node whose first Wikidata class in additionalType
// maps, in the classtype table the research tools type entities with, to a
// strict subtype of its type: a Thing or Place whose class is city is a City.
// Only the first class counts: it is an instance-of class, while a later one
// may be a subclass-of class. An item that is itself a class (a kind of
// thing) stays a Thing however, which the draft does not show, so this is a
// warning to weigh, not an error. A node whose own item, in sameAs or a
// Wikidata identifier, is a class in the table is a kind of thing for sure,
// and only the types whose items are always classes apply to it. It tells
// the node in the retype ledger.
type ClassType struct{}

// ID is the check's ID.
func (ClassType) ID() string { return "graph/class-type" }

// classed is a node, its one type, its first Wikidata class and the subtype
// that class gives it.
type classed struct {
	jsonld.Node
	ty, class, want string
}

// find is every node whose first Wikidata class gives it a subtype.
func (ClassType) find(c *audit.Context) []classed {
	var out []classed
	for _, l := range c.Nodes {
		ty := c.Single(l.N)
		if ty == "" {
			continue
		}
		classes := jsonld.Strs(l.N, "additionalType")
		if len(classes) == 0 {
			continue
		}
		m := jsonld.WikidataItem.FindStringSubmatch(classes[0])
		if m == nil {
			continue
		}
		// another class that gives the node its type backs it: the Blue
		// Mountains are a mountain range and a region, so a Place
		if slices.ContainsFunc(classes[1:], func(cl string) bool {
			o := jsonld.WikidataItem.FindStringSubmatch(cl)
			return o != nil && classtype.Of([]string{o[1]}, false) == ty
		}) {
			continue
		}
		kind := slices.ContainsFunc(jsonld.OwnItems(l.N), func(q string) bool { return classtype.Of([]string{q}, false) != "" })
		if want := classtype.Of([]string{m[1]}, kind); want != "" && want != ty && slices.Contains(c.Vocab.V.Ancestors(want), ty) {
			out = append(out, classed{l, ty, m[1], want})
		}
	}
	return out
}

// Run reports each node with the subtype its class gives it.
func (k ClassType) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range k.find(c) {
		also := ""
		if f.want == "LandmarksOrHistoricalBuildings" {
			also = ", with TouristAttraction too when visitors go to see it"
		}
		out = append(out, audit.Issuef(f.Path, audit.Warning, "%s with the Wikidata class %s: type it %s%s, unless the item is a kind of thing rather than one", f.ty, f.class, f.want, also))
	}
	return out
}

// Retypes tells each node its class gives a subtype, and names it.
func (k ClassType) Retypes(c *audit.Context, _ audit.Retyped) []audit.Retype {
	var out []audit.Retype
	for _, f := range k.find(c) {
		out = append(out, audit.Retype{Node: f.N, Named: []string{f.want}})
	}
	return out
}
