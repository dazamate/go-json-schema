package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestSupersededType(t *testing.T) {
	expectTerms(t, terms.SupersededType{}, []termCase{
		{"a superseded type", `{"@type":"Taxi","name":"Harbour Cabs"}`, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "type Taxi is superseded by TaxiService"},
		}},
		{"prefixed, beside another", `{"@type":["schema:Taxi","Code"]}`, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "type Taxi is superseded by TaxiService"},
			{Path: "$", Severity: audit.Warning, Message: "type Code is superseded by SoftwareSourceCode"},
		}},
		{"a current type", `{"@type":"TaxiService","name":"Harbour Cabs"}`, nil},
		// an unknown type is only unknown
		{"an unknown type", `{"@type":"Taxii"}`, nil},
	})
}
