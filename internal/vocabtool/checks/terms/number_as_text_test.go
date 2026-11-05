package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

func TestNumberAsText(t *testing.T) {
	expectTerms(t, terms.NumberAsText{}, []termCase{
		{"numbers written as text", `{"@type":"Article","wordCount":" 900 ","commentCount":["3","many",4]}`, []audit.Issue{
			{Path: "$.wordCount", Severity: audit.Warning, Message: `wordCount takes a number: write 900, not " 900 "`},
			{Path: "$.commentCount", Severity: audit.Warning, Message: `commentCount takes a number: write 3, not "3"`},
		}},
		{"a number", `{"@type":"Article","wordCount":900}`, nil},
		// a property that also takes text may hold a number as text
		{"a property that takes text", `{"@type":"Product","sku":"900"}`, nil},
	})
}
