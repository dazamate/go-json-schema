package terms_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)

// termCase is a terms check's test case: a draft, checked alone, and what the
// check finds in it.
type termCase struct {
	name  string
	draft string
	want  []audit.Issue
}

// expectTerms runs each case through the check.
func expectTerms(t *testing.T, c audit.Check, cases []termCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, c, audittest.Input{Draft: tc.draft}, tc.want) })
	}
}

func TestUnknownType(t *testing.T) {
	expectTerms(t, terms.UnknownType{}, []termCase{
		{"a misspelt type", `{"@type":"Recipee","name":"x"}`, []audit.Issue{
			{Path: "$", Severity: audit.Error, Message: `unknown type "Recipee"; did you mean Recipe?`},
		}},
		{"a type nothing is near", `{"@type":"Zzqxwvbk"}`, []audit.Issue{
			{Path: "$", Severity: audit.Error, Message: `unknown type "Zzqxwvbk"`},
		}},
		{"a node of a graph", `[{"@type":"Recipe","offers":{"@type":"Offer","availability":"Nope"}},{"@type":"Persn"}]`, []audit.Issue{
			{Path: "$.@graph[1]", Severity: audit.Error, Message: `unknown type "Persn"; did you mean Person?`},
		}},
		{"known types, prefixed", `{"@type":["schema:Recipe","https://schema.org/HowTo"],"schema:name":"x","author":{"@id":"#me"}}`, nil},
		// the walk goes no further into a key that is no schema.org property
		{"under an unknown property", `{"@type":"Product","fooBar":{"@type":"Persn","availability":"Nope"}}`, nil},
		{"under a Google annotation", `{"@type":"SearchAction","query-input":{"@type":"Persn","availability":"Nope"},"target-output":"x"}`, nil},
		{"under a foreign key", `{"@type":"Offer","ex:offers":{"@type":"Persn"}}`, nil},
	})
}
