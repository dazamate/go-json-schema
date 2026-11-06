package graph

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(PersonName{}) }

// PersonName warns about a Person named with a given and a family name that
// does not give them apart. It needs the page's facts but not an indexed
// page: the markup's own faults are held to a page search engines skip too.
type PersonName struct{}

// ID is the check's ID.
func (PersonName) ID() string { return "graph/person-name" }

// personName matches a given name, an optional middle name and a family name.
var personName = regexp.MustCompile(`^(\p{Lu}[\p{L}'’-]+)(?:\s+\p{Lu}[\p{L}'’-]+)?\s+(\p{Lu}[\p{L}'’-]+)$`)

// Run reports each Person whose name splits in two and who has neither part.
func (PersonName) Run(c *audit.Context) []audit.Issue {
	if c.Page == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		if !c.Is(l.N, "Person") {
			continue
		}
		_, g := l.N.Get("givenName")
		_, f := l.N.Get("familyName")
		for _, s := range jsonld.Strs(l.N, "name") {
			if m := personName.FindStringSubmatch(strings.TrimSpace(s)); m != nil && !g && !f {
				out = append(out, audit.Issuef(l.Path, audit.Warning, "this Person is named %q: set givenName %s and familyName %s", s, m[1], m[2]))
			}
			break
		}
	}
	return out
}
