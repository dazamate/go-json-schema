package terms

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(EnumMember{}) }

// EnumMember errors on a value that is not a member of the enumeration its
// property takes as its only range, such as availability "InStok", and lists
// the members it allows. A member's name or URL is a member: the draft's bare
// names are expanded to their URL before any check runs.
type EnumMember struct{}

// ID is the check's ID.
func (EnumMember) ID() string { return "terms/enum-member" }

// Run reports each value that is no member, with up to 15 allowed members.
func (EnumMember) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, v := range walkOf(c).values {
		enum, only := c.Vocab.EnumOnly(v.prop)
		if !only {
			continue
		}
		allowed := c.Vocab.Members(enum)
		if slices.Contains(allowed, jsonld.StripPrefix(v.text)) {
			continue
		}
		if len(allowed) > 15 {
			allowed = append(allowed[:15:15], "...")
		}
		out = append(out, audit.Issuef(v.path, audit.Error, "%q is not an %s member; allowed: %s", v.text, enum, strings.Join(allowed, ", ")))
	}
	return out
}
