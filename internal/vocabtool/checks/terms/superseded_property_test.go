package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestSupersededProperty(t *testing.T) {
	expectTerms(t, terms.SupersededProperty{}, []termCase{
		{"a superseded property", `{"@type":"LocalBusiness","map":"https://maps.test/x"}`, []audit.Issue{
			{Path: "$.map", Severity: audit.Warning, Message: "property map is superseded by hasMap"},
		}},
		{"prefixed, on a nested node", `{"@type":"WebPage","about":{"@type":"Organization","schema:employees":{"@type":"Person","name":"Sam"}}}`, []audit.Issue{
			{Path: "$.about.schema:employees", Severity: audit.Warning, Message: "property employees is superseded by employee"},
		}},
		{"a current property", `{"@type":"LocalBusiness","hasMap":"https://maps.test/x"}`, nil},
	})
}
