package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestUnknownProperty(t *testing.T) {
	expectTerms(t, terms.UnknownProperty{}, []termCase{
		{"a miscased property", `{"@type":"Recipe","recipeingredient":["1 lemon"]}`, []audit.Issue{
			{Path: "$.recipeingredient", Severity: audit.Error, Message: `unknown property "recipeingredient"; did you mean recipeIngredient?`},
		}},
		{"a property nothing is near", `{"@type":"Thing","zzqxwvbkjq":"x"}`, []audit.Issue{
			{Path: "$.zzqxwvbkjq", Severity: audit.Error, Message: `unknown property "zzqxwvbkjq"`},
		}},
		{"on a nested node, and the node under it", `{"@type":"Product","offers":{"@type":"Offer","pricee":"0","fooBar":{"@type":"Offer","pricee":"0"}}}`, []audit.Issue{
			{Path: "$.offers.pricee", Severity: audit.Error, Message: `unknown property "pricee"; did you mean price?`},
			{Path: "$.offers.fooBar", Severity: audit.Error, Message: `unknown property "fooBar"`},
		}},
		{"a known property", `{"@type":"Recipe","recipeIngredient":["1 lemon"]}`, nil},
		// a Google annotation and a key that is not a schema.org term are not
		// unknown properties
		{"a Google annotation", `{"@type":"SearchAction","query-input":"required name=q"}`, nil},
		{"a foreign key", `{"@type":"Thing","ex:fooBar":"x"}`, nil},
	})
}
