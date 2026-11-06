package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestOutsideDomain(t *testing.T) {
	expectTerms(t, terms.OutsideDomain{}, []termCase{
		// openingHours and articleBody are their subtypes' properties instead
		{"a property no subtype takes", subtypeGraph, []audit.Issue{
			{Path: "$.@graph[2].knowsAbout", Severity: audit.Info, Message: "knowsAbout is not listed for Recipe (domains: Organization, Person); kept, since schema.org domains are guidelines rather than rules"},
		}},
		{"a property its type takes", `{"@type":"Person","knowsAbout":"Lemon"}`, nil},
		{"an untyped node", `{"@type":"WebPage","about":{"name":"Tart","knowsAbout":"Lemon"}}`, nil},
		{"an unknown type", `{"@type":"Recipee","knowsAbout":"Lemon"}`, nil},
	})
}
