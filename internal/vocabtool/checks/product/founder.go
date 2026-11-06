package product

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(Founder{}) }

var foundedBy = regexp.MustCompile(`(?:[Ff]ounded|[Ss]tarted|[Ee]stablished)[^.]{0,40}? by ([A-Z][a-z]+ [A-Z][a-z]+)`)

// Founder warns when a page names who founded the business and no node has a
// founder.
type Founder struct{}

// ID is the check's ID.
func (Founder) ID() string { return "product/founder" }

// Run reports the first founder the page names.
func (Founder) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || jsonld.HasKey(c.Nodes, "founder") {
		return nil
	}
	m := foundedBy.FindStringSubmatch(c.Page.Main)
	if m == nil {
		return nil
	}
	given, family, _ := strings.Cut(m[1], " ")
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page names the business's founder, %s: set its founder, a Person (name, givenName %s, familyName %s)", m[1], given, family)}
}
