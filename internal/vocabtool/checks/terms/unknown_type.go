package terms

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

func init() { audit.Register(UnknownType{}) }

// UnknownType errors on a @type name the vocabulary lacks, naming the type
// it was likely meant to be: no search engine reads a node of a type that
// does not exist.
type UnknownType struct{}

// ID is the check's ID.
func (UnknownType) ID() string { return "terms/unknown-type" }

// Run reports each unknown type name.
func (UnknownType) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, t := range walkOf(c).types {
		if t.class == nil {
			out = append(out, unknown(c, t.path, vocab.KindType, t.name))
		}
	}
	return out
}
