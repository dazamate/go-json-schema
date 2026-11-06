package terms

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(SupersededProperty{}) }

// SupersededProperty warns on a property schema.org has replaced, naming the
// properties that replace it, such as map for hasMap.
type SupersededProperty struct{}

// ID is the check's ID.
func (SupersededProperty) ID() string { return "terms/superseded-property" }

// Run reports each superseded property.
func (SupersededProperty) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, k := range walkOf(c).props {
		if len(k.prop.SupersededBy) > 0 {
			out = append(out, audit.Issuef(k.path, audit.Warning, "property %s is superseded by %s", k.term, strings.Join(k.prop.SupersededBy, ", ")))
		}
	}
	return out
}
