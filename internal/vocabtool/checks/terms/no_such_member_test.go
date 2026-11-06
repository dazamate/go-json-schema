package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestNoSuchMember(t *testing.T) {
	expectTerms(t, terms.NoSuchMember{}, []termCase{
		// itemListOrder takes text or an ItemListOrderType: only a schema.org
		// URL claims to be a member
		{"a schema.org URL no member has", `{"@type":"ItemList","itemListOrder":"https://schema.org/Ascending"}`, []audit.Issue{
			{Path: "$.itemListOrder", Severity: audit.Warning, Message: `"https://schema.org/Ascending" names no schema.org enumeration member`},
		}},
		{"text and a member", `{"@type":"ItemList","itemListOrder":["Ascending","https://schema.org/ItemListOrderAscending"]}`, nil},
		// a property that takes only an enumeration gets the member error
		{"an enumeration's only property", `{"@type":"Offer","availability":"https://schema.org/Nope"}`, nil},
		// a URL that names no member is a value like any other where no
		// enumeration is a range
		{"no enumeration in the range", `{"@type":"Thing","url":"https://schema.org/Nope"}`, nil},
	})
}
