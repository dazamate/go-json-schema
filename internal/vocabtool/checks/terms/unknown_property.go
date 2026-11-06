package terms

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

func init() { audit.Register(UnknownProperty{}) }

// UnknownProperty errors on a key that names no schema.org property, naming
// the property it was likely meant to be: search engines drop what it holds.
// Google's annotations and other vocabularies' terms are not unknown.
type UnknownProperty struct{}

// ID is the check's ID.
func (UnknownProperty) ID() string { return "terms/unknown-property" }

// Run reports each unknown property.
func (UnknownProperty) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, k := range walkOf(c).unknown {
		out = append(out, unknown(c, k.path, vocab.KindProperty, k.name))
	}
	return out
}
