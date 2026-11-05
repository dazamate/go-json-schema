package terms

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(OutsideDomain{}) }

// OutsideDomain notes a property used on a type outside its listed domains,
// when no subtype of the type takes it either. It is only info, and the
// property is kept: schema.org calls domains "closer to guidelines than
// formal rules" (https://schema.org/docs/datamodel.html).
type OutsideDomain struct{}

// ID is the check's ID.
func (OutsideDomain) ID() string { return "terms/outside-domain" }

// Run notes each such property with the types it is listed for.
func (OutsideDomain) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, k := range walkOf(c).props {
		// a subtype that takes it is the subtype check's to name
		if !k.unlisted(c.Vocab) || len(c.Vocab.SubtypesTaking(k.types, k.term)) > 0 {
			continue
		}
		out = append(out, audit.Issuef(k.path, audit.Info, "%s is not listed for %s (domains: %s); kept, since schema.org domains are guidelines rather than rules", k.term, strings.Join(k.types, ", "), strings.Join(k.prop.Domains, ", ")))
	}
	return out
}
