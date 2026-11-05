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
		// oliverbonas.com and the WordPress SEO plugins: a crumb's item is
		// its URL and name, which Google reads as the crumb
		{"a breadcrumb's item", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"item":{"@id":"https://www.oliverbonas.com/returns-and-refunds","name":"Returns and Refunds"}}]}`, nil},
		// instaelectricians.com: the author names a node the graph types
		{"a named reference to a typed node", `{"@context":"https://schema.org","@graph":[
			{"@type":"Organization","@id":"https://www.instaelectricians.com/blog/electrical-panel-upgrade-cost/#author","name":"instaelectricians"},
			{"@type":"BlogPosting","headline":"Electrical Panel Upgrade Cost","author":{"@id":"https://www.instaelectricians.com/blog/electrical-panel-upgrade-cost/#author","name":"instaelectricians"}}]}`, nil},
		// an ItemList's item is a thing the list holds, not a crumb
		{"an item list's item", `{"@type":"ItemList","itemListElement":[{"@type":"ListItem","position":1,"item":{"name":"Gold locket","url":"https://shop.test/products/gold-locket"}}]}`, []audit.Issue{
			{Path: "$.itemListElement[0].item", Severity: audit.Warning, Message: "node has properties but no @type"},
		}},
	})
}
