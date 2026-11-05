package terms

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(GoogleAnnotation{}) }

// GoogleAnnotation notes a key such as query-input that Google's action
// markup adds beside a schema.org Action. It is not a schema.org property,
// but Google reads it, so it is kept as written and not checked further.
type GoogleAnnotation struct{}

// ID is the check's ID.
func (GoogleAnnotation) ID() string { return "terms/google-annotation" }

// Run notes each annotation.
func (GoogleAnnotation) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, k := range walkOf(c).annotations {
		out = append(out, audit.Issuef(k.path, audit.Info, "%s is Google's action annotation, not a schema.org property; kept", k.name))
	}
	return out
}
