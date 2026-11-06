package terms

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(ForeignTerm{}) }

// ForeignTerm warns on a key from another vocabulary, a prefixed name or a
// full URL outside schema.org: the vocabulary cannot check it, so it is left
// as written and not checked further.
type ForeignTerm struct{}

// ID is the check's ID.
func (ForeignTerm) ID() string { return "terms/foreign-term" }

// Run reports each foreign key.
func (ForeignTerm) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, k := range walkOf(c).foreign {
		out = append(out, audit.Issuef(k.path, audit.Warning, "%q is not a schema.org term; left as is", k.name))
	}
	return out
}
