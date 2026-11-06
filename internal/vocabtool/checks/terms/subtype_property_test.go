package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

// subtypeGraph holds properties a node's type does not list: two that its
// subtypes do, and one no subtype of it takes.
const subtypeGraph = `{"@context":"https://schema.org","@graph":[
	{"@type":"Organization","name":"Acme","openingHours":"Mo-Fr 09:00-17:00"},
	{"@type":"CreativeWork","name":"Notes","articleBody":"Text."},
	{"@type":"Recipe","name":"Tart","knowsAbout":"Lemon"}]}`

func TestSubtypeProperty(t *testing.T) {
	expectTerms(t, terms.SubtypeProperty{}, []termCase{
		// both subtypes that take it are named, the most general ones only
		{"properties subtypes take", subtypeGraph, []audit.Issue{
			{Path: "$.@graph[0].openingHours", Severity: audit.Warning, Message: "openingHours is not listed for Organization but is for its subtypes EducationalOrganization and LocalBusiness: type it EducationalOrganization or LocalBusiness"},
			{Path: "$.@graph[1].articleBody", Severity: audit.Warning, Message: "articleBody is not listed for CreativeWork but is for its subtype Article: type it Article"},
		}},
		{"a property its type takes", `{"@type":"LocalBusiness","openingHours":"Mo-Fr 09:00-17:00"}`, nil},
		// with no known type there is no subtype to name
		{"an untyped node", `{"@type":"WebPage","about":{"name":"Acme","openingHours":"Mo-Fr"}}`, nil},
		{"an unknown type", `{"@type":"Organizaton","openingHours":"Mo-Fr"}`, nil},
	})
}
