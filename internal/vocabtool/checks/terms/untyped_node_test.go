package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestUntypedNode(t *testing.T) {
	expectTerms(t, terms.UntypedNode{}, []termCase{
		// a node is reported once its properties are walked: the inner first
		{"nested untyped nodes", `{"@type":"Recipe","author":{"name":"Jane","knows":{"name":"Sam"}}}`, []audit.Issue{
			{Path: "$.author.knows", Severity: audit.Warning, Message: "node has properties but no @type"},
			{Path: "$.author", Severity: audit.Warning, Message: "node has properties but no @type"},
		}},
		{"a node of a graph", `{"@graph":[{"availability":"InStock"}]}`, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: "node has properties but no @type"},
		}},
		{"a reference", `{"@type":"Recipe","author":{"@id":"#me"}}`, nil},
		{"a typed node", `{"@type":"Recipe","author":{"@type":"Person","name":"Jane"}}`, nil},
		{"under an unknown property", `{"@type":"Product","fooBar":{"name":"x"}}`, nil},
	})
}
