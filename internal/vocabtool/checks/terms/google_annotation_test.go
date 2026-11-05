package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestGoogleAnnotation(t *testing.T) {
	expectTerms(t, terms.GoogleAnnotation{}, []termCase{
		{"a site search box", `{"@type":"WebSite","url":"https://x.test/","potentialAction":{"@type":"SearchAction",
			"target":{"@type":"EntryPoint","urlTemplate":"https://x.test/search?q={search_term_string}"},"query-input":"required name=search_term_string"}}`, []audit.Issue{
			{Path: "$.potentialAction.query-input", Severity: audit.Info, Message: "query-input is Google's action annotation, not a schema.org property; kept"},
		}},
		{"two, one holding a node", `{"@type":"SearchAction","query-input":{"@type":"Persn","availability":"Nope"},"target-output":"x"}`, []audit.Issue{
			{Path: "$.query-input", Severity: audit.Info, Message: "query-input is Google's action annotation, not a schema.org property; kept"},
			{Path: "$.target-output", Severity: audit.Info, Message: "target-output is Google's action annotation, not a schema.org property; kept"},
		}},
		{"no annotation", `{"@type":"SearchAction","target":"https://x.test/search?q={q}"}`, nil},
	})
}
