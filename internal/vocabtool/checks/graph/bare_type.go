package graph

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(BareType{}) }

// BareType warns about a node typed only as one of the types that name
// nothing more than a category, and names the subtypes that say what the node
// is.
type BareType struct{}

// ID is the check's ID.
func (BareType) ID() string { return "graph/bare-type" }

// bareTypes name nothing more than a category; a node is one of their
// subtypes.
var bareTypes = map[string]string{
	"Intangible":      "Service, Offer, DefinedTerm, Audience",
	"StructuredValue": "PropertyValue, QuantitativeValue, MonetaryAmount, ContactPoint",
	"Action":          "AskAction, ReadAction, SearchAction, QuoteAction",
	"MediaObject":     "ImageObject, VideoObject, AudioObject",
	"Enumeration":     "a member of the enumeration",
}

// Run reports each node typed only as a bare type.
func (BareType) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, l := range c.Nodes {
		if types := jsonld.TypeNames(l.N); len(types) == 1 {
			if use, ok := bareTypes[types[0]]; ok {
				out = append(out, audit.Issuef(l.Path, audit.Warning, "a bare %s says little: use the subtype that fits (%s)", types[0], use))
			}
		}
	}
	return out
}
