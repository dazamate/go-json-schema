package graph

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
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

var (
	// honorific is a title before the name, suffix a generation or a degree after it.
	honorific = regexp.MustCompile(`^(?:Dr|Prof|Mr|Mrs|Ms|Mx|Sir|Dame|Rev)\.?\s+`)
	suffix    = regexp.MustCompile(`[,\s]+(?:Jr|Sr|II|III|PhD|MD)\.?$`)
)

// bareName is a name without its honorific and suffix, so "Dr Jane Smith" is
// Jane Smith and "John Smith Jr" is John Smith.
func bareName(s string) string {
	s = strings.TrimSpace(s)
	for {
		t := suffix.ReplaceAllString(honorific.ReplaceAllString(s, ""), "")
		if t == s {
			return s
		}
		s = t
	}
}

// businessWord is a word a business's name holds and a person's does not:
// "Love It Landscaping", "The Bread Kitchen", "Made For You".
var businessWord = regexp.MustCompile(`\b(?:The|It|Of|And|For|At|With|Our|Your)\b`)

// Run reports each Person whose name splits in two and who has neither part.
// A Person named for an organization or brand of the graph ("Allrecipes Test
// Kitchen"), or whose name holds a word no person's name holds ("Love It
// Landscaping"), is the business, not a person with a given name. A site
// named after its person ("Jane Smith"), and the site's person that Yoast and
// Rank Math type Person and Organization, still are people.
func (PersonName) Run(c *audit.Context) []audit.Issue {
	if c.Page == nil {
		return nil
	}
	var orgs []string
	for _, l := range c.Nodes {
		if !c.Is(l.N, "Person") && (c.Is(l.N, "Organization") || c.Is(l.N, "Brand")) {
			for _, s := range jsonld.Strs(l.N, "name") {
				if w := txt.NormWords(s); w != "" {
					orgs = append(orgs, " "+w+" ")
				}
			}
		}
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		if !c.Is(l.N, "Person") {
			continue
		}
		_, g := l.N.Get("givenName")
		_, f := l.N.Get("familyName")
		for _, s := range jsonld.Strs(l.N, "name") {
			name := " " + txt.NormWords(s) + " "
			if businessWord.MatchString(s) || slices.ContainsFunc(orgs, func(o string) bool { return strings.Contains(name, o) }) {
				break
			}
			if m := personName.FindStringSubmatch(bareName(s)); m != nil && !g && !f {
				out = append(out, audit.Issuef(l.Path, audit.Warning, "this Person is named %q: set givenName %s and familyName %s", s, m[1], m[2]))
			}
			break
		}
	}
	return out
}
