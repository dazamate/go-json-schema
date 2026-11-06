package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestProductName(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"a Product with no name", `{"@type":"Product","offers":{"@type":"Offer","price":1}}`, []audit.Issue{
			warn("$", `Missing field "name": give this Product the name the page shows for it`),
		}},
		{"a Vehicle is read as a Product", `{"@type":"ItemList","itemListElement":[{"@type":"Vehicle","name":"10 ft"},{"@type":"Vehicle","fuelType":"diesel"}]}`, []audit.Issue{
			warn("$.itemListElement[1]", `Missing field "name": give this Vehicle the name the page shows for it`),
		}},
		{"a ProductGroup with no name", `{"@type":"ProductGroup","productGroupID":"box"}`, []audit.Issue{
			warn("$", `Missing field "name": give this ProductGroup the name the page shows for it`),
		}},
		{"a named Product", `{"@type":"Product","name":"Small moving box"}`, nil},
		{"a reference is checked where its node is", `{"@type":"WebPage","name":"Boxes","about":{"@type":"Product","@id":"https://example.com/#box"}}`, nil},
		{"a node that is no Product", `{"@type":"Service"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.ProductName{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
