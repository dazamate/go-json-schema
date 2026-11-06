package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestPersonName(t *testing.T) {
	const draft = `{"@context":"https://schema.org","@type":"Article","headline":"Tea","author":{"@type":"Person","name":"Ann Lee"}}`
	fires := []audit.Issue{{Path: "$.author", Severity: audit.Warning, Message: `this Person is named "Ann Lee": set givenName Ann and familyName Lee`}}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"a given and a family name", draft, audittest.Input{PageHTML: teaPage}, fires},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, fires},
		// the markup's own faults are held to a page search engines skip too
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"a middle name", `{"@type":"Person","name":"Ann Marie Lee"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `this Person is named "Ann Marie Lee": set givenName Ann and familyName Lee`},
		}},
		{"givenName set", `{"@type":"Person","name":"Ann Lee","givenName":"Ann"}`, audittest.Input{PageText: "Tea."}, nil},
		{"familyName set", `{"@type":"Person","name":"Ann Lee","familyName":"Lee"}`, audittest.Input{PageText: "Tea."}, nil},
		{"one word", `{"@type":"Person","name":"Ann"}`, audittest.Input{PageText: "Tea."}, nil},
		{"not capitalised", `{"@type":"Person","name":"ann lee"}`, audittest.Input{PageText: "Tea."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, graph.PersonName{}, tc.in, tc.want)
		})
	}
}
