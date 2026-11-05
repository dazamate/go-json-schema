package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestUnnamedEntity(t *testing.T) {
	const draft = `{"@context":"https://schema.org","@graph":[
		{"@type":"City","@id":"https://x.test/#sydney","name":"Sydney"},
		{"@type":"Occupation","@id":"https://x.test/#chef","name":"Chef"},
		{"@type":"WebSite","@id":"https://x.test/#website","about":[{"@type":"Thing","name":"Catering"}]},
		{"@type":"WebPage","@id":"https://x.test/#webpage",
		 "about":[{"@type":"Thing","name":"Toast (honor)"},{"@type":"Thing","name":"Bride"},{"@id":"https://x.test/#sydney"},
		          {"@type":"Thing","name":"Groomsman","alternateName":["groomsmen"]},{"@type":"Thing","name":"Hajong marriage"}],
		 "mentions":[{"@type":"Thing","name":"Pre-wedding customs"},{"@id":"https://x.test/#chef"},"Kitchen",{"@id":"https://elsewhere.test/#x"}]}]}`
	const text = "Our Sydney MC gives the toast, guides the brides and the groomsmen through pre wedding customs."
	fires := []audit.Issue{
		{Path: "$.@graph[3].about[4]", Severity: audit.Warning, Message: "Hajong marriage is not named on the page: only what the page shows goes in about"},
		{Path: "$.@graph[3].mentions[1]", Severity: audit.Warning, Message: "Chef is not named on the page: only what the page shows goes in mentions"},
		{Path: "$.@graph[3].mentions[2]", Severity: audit.Warning, Message: "Kitchen is not named on the page: only what the page shows goes in mentions"},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"entities the page does not name", draft, audittest.Input{PageText: text}, fires},
		// the check holds the markup to a page search engines skip too
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(teaPage), PageText: text}, fires},
		{"a single value", `{"@type":"WebPage","mentions":"Kettle"}`, audittest.Input{PageText: text}, []audit.Issue{
			{Path: "$.mentions", Severity: audit.Warning, Message: "Kettle is not named on the page: only what the page shows goes in mentions"},
		}},
		{"the page's text from its HTML", `{"@type":"WebPage","about":[{"@type":"Thing","name":"Boil"},{"@type":"Thing","name":"Kitchen"}]}`, audittest.Input{PageHTML: teaPage}, []audit.Issue{
			{Path: "$.about[1]", Severity: audit.Warning, Message: "Kitchen is not named on the page: only what the page shows goes in about"},
		}},
		{"every entity named", `{"@type":"WebPage","about":[{"@type":"Thing","name":"Tea ceremony"}]}`, audittest.Input{PageText: "The tea ceremony."}, nil},
		{"no entities", `{"@type":"WebPage","name":"Tea"}`, audittest.Input{PageText: text}, nil},
		{"no page", draft, audittest.Input{}, nil},
		{"blank page text", draft, audittest.Input{PageText: "  \n "}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.UnnamedEntity{}, tc.in, tc.want)
		})
	}
}
