package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestEnumMember(t *testing.T) {
	expectTerms(t, terms.EnumMember{}, []termCase{
		// a member name, expanded to its URL before any check runs, is a member
		{"one value of a list", `{"@type":"Offer","availability":["InStock","InStok"],"itemCondition":"https://schema.org/NewCondition"}`, []audit.Issue{
			{Path: "$.availability[1]", Severity: audit.Error, Message: `"InStok" is not an ItemAvailability member; allowed: BackOrder, Discontinued, InStock, InStoreOnly, LimitedAvailability, MadeToOrder, OnlineOnly, OutOfStock, PreOrder, PreSale, Reserved, SoldOut`},
		}},
		{"a nested node", `[{"@type":"Recipe","offers":{"@type":"Offer","availability":"Nope"}},{"@type":"Persn"}]`, []audit.Issue{
			{Path: "$.@graph[0].offers.availability", Severity: audit.Error, Message: `"Nope" is not an ItemAvailability member; allowed: BackOrder, Discontinued, InStock, InStoreOnly, LimitedAvailability, MadeToOrder, OnlineOnly, OutOfStock, PreOrder, PreSale, Reserved, SoldOut`},
		}},
		{"a short enumeration", `{"@type":"Offer","itemCondition":"Mint"}`, []audit.Issue{
			{Path: "$.itemCondition", Severity: audit.Error, Message: `"Mint" is not an OfferItemCondition member; allowed: DamagedCondition, NewCondition, RefurbishedCondition, UsedCondition`},
		}},
		{"members by name and URL", `{"@type":"Offer","availability":["InStock","http://schema.org/OutOfStock","schema:PreOrder"]}`, nil},
		// the walk checks only a known property's values
		{"under an unknown property", `{"@type":"Product","fooBar":{"@type":"Offer","availability":"Nope"}}`, nil},
		{"under a Google annotation", `{"@type":"SearchAction","query-input":{"@type":"Offer","availability":"Nope"}}`, nil},
	})
}
