package terms

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(NoSuchMember{}) }

// NoSuchMember warns on a schema.org URL that names no enumeration member,
// given to a property that takes an enumeration beside other values, such as
// itemListOrder's Text or ItemListOrderType. Only a schema.org URL claims to
// be a member there, so only that is checked; a property that takes only an
// enumeration is the member check's.
type NoSuchMember struct{}

// ID is the check's ID.
func (NoSuchMember) ID() string { return "terms/no-such-member" }

// Run reports each such URL.
func (NoSuchMember) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, v := range walkOf(c).values {
		if _, only := c.Vocab.EnumOnly(v.prop); only {
			continue
		}
		name, isURL := jsonld.CutSchemaURL(v.text)
		if !isURL {
			continue
		}
		if _, isMember := c.Vocab.V.Members[name]; !isMember && c.Vocab.RangesEnum(v.prop) {
			out = append(out, audit.Issuef(v.path, audit.Warning, "%q names no schema.org enumeration member", v.text))
		}
	}
	return out
}
