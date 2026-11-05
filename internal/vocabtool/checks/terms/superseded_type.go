package terms

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(SupersededType{}) }

// SupersededType warns on a type schema.org has replaced, naming the types
// that replace it: the old one still parses, but its successor is what the
// vocabulary describes now.
type SupersededType struct{}

// ID is the check's ID.
func (SupersededType) ID() string { return "terms/superseded-type" }

// Run reports each superseded type.
func (SupersededType) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, t := range walkOf(c).types {
		if t.class != nil && len(t.class.SupersededBy) > 0 {
			out = append(out, audit.Issuef(t.path, audit.Warning, "type %s is superseded by %s", t.term, strings.Join(t.class.SupersededBy, ", ")))
		}
	}
	return out
}
