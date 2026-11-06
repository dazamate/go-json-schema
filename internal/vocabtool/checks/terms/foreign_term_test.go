package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestForeignTerm(t *testing.T) {
	expectTerms(t, terms.ForeignTerm{}, []termCase{
		{"prefixed and full keys", `{"@type":"Offer","ex:availability":"InStock","https://example.org/terms/size":"L","ex:offers":{"@type":"Persn"}}`, []audit.Issue{
			{Path: "$.ex:availability", Severity: audit.Warning, Message: `"ex:availability" is not a schema.org term; left as is`},
			{Path: "$.https://example.org/terms/size", Severity: audit.Warning, Message: `"https://example.org/terms/size" is not a schema.org term; left as is`},
			{Path: "$.ex:offers", Severity: audit.Warning, Message: `"ex:offers" is not a schema.org term; left as is`},
		}},
		{"schema.org keys, prefixed", `{"@type":"Offer","schema:availability":"InStock","https://schema.org/price":"10"}`, nil},
		{"under an unknown property", `{"@type":"Product","fooBar":{"@type":"Offer","ex:size":"L"}}`, nil},
	})
}
