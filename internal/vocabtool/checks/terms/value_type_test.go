package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestValueType(t *testing.T) {
	expectTerms(t, terms.ValueType{}, []termCase{
		// validator.schema.org: INVALID_OBJECT hasPart,ItemList,CreativeWork
		{"a list in hasPart", `{"@type":"WebPage","hasPart":{"@type":"ItemList","numberOfItems":3}}`, []audit.Issue{
			{Path: "$.hasPart", Severity: audit.Error, Message: `hasPart takes CreativeWork, not ItemList: give it a node of a type it takes, or move the ItemList to a property that takes it`},
		}},
		{"a reference to a list in hasPart", `{"@context":"https://schema.org","@graph":[
			{"@type":"WebPage","@id":"https://x.test/#webpage","hasPart":[{"@id":"https://x.test/#nav"},{"@id":"https://x.test/#guide"}]},
			{"@type":"SiteNavigationElement","@id":"https://x.test/#nav","cssSelector":"nav"},
			{"@type":"ItemList","@id":"https://x.test/#guide","numberOfItems":3}]}`, []audit.Issue{
			{Path: "$.@graph[0].hasPart[1]", Severity: audit.Error, Message: `hasPart takes CreativeWork, not ItemList: give it a node of a type it takes, or move the ItemList to a property that takes it`},
		}},
		{"a list on its block", `{"@type":"WebPage","hasPart":{"@type":"SiteNavigationElement","cssSelector":"nav","mainEntity":{"@type":"ItemList","numberOfItems":3}}}`, nil},
		{"an author that is a place", `{"@type":"Article","author":{"@type":"Place","name":"Boston"}}`, []audit.Issue{
			{Path: "$.author", Severity: audit.Error, Message: `author takes Organization or Person, not Place: give it a node of a type it takes, or move the Place to a property that takes it`},
		}},
		{"a subtype of the type it takes", `{"@type":"Product","offers":{"@type":"AggregateOffer","lowPrice":9}}`, nil},
		{"one of a node's types fits", `{"@type":"Article","author":{"@type":["Place","Organization"],"name":"Boston Movers"}}`, nil},
		// a Role stands in for any value
		{"a Role", `{"@type":"SportsTeam","athlete":{"@type":"OrganizationRole","roleName":"captain","athlete":{"@type":"Person","name":"A"}}}`, nil},
		// a property that takes only text and numbers is the literal checks'
		{"a node where text goes", `{"@type":"Thing","name":{"@type":"Thing","name":"x"}}`, nil},
		{"a value of an unknown type", `{"@type":"WebPage","hasPart":{"@type":"Lisst"}}`, nil},
		{"an untyped value", `{"@type":"WebPage","hasPart":{"name":"x"}}`, nil},
		{"a reference the draft does not define", `{"@type":"WebPage","hasPart":{"@id":"https://x.test/#elsewhere"}}`, nil},
	})
}
