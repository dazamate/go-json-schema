package keyword_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

// loosePage is a flat graph whose page points at a Piano node and a Truck
// node that stand at the top.
const loosePage = `{"@context":"https://schema.org","@graph":[
 {"@id":"https://a.example/#org","@type":"OnlineBusiness","name":"Lift My Boxes"},
 {"@id":"https://a.example/#website","@type":"WebSite","publisher":{"@id":"https://a.example/#org"}},
 {"@id":"https://a.example/p/#webpage","@type":"WebPage","url":"https://a.example/p/","about":[{"@id":"https://a.example/#piano"}],"mentions":[{"@id":"https://a.example/#truck"}]},
 {"@id":"https://a.example/#piano","@type":"Thing","name":"Piano"},
 {"@id":"https://a.example/#truck","@type":"Thing","headline":"Truck"},
 {"@id":"https://a.example/#unused","@type":"Thing","name":"Unused"}]}`

const looseTail = "validator.schema.org lists each top-level node as an item of its own, so they read as loose things with no parent. Nest each where it is first referenced, keeping its @id, and leave only the page, the WebSite and the business at the top"

func TestLooseNodes(t *testing.T) {
	// ten Things the page points at: the message names eight
	var nodes, refs []string
	for i := 0; i < 10; i++ {
		nodes = append(nodes, fmt.Sprintf(`{"@id":"https://a.example/#t%d","@type":"Thing","name":"Thing %d"}`, i, i))
		refs = append(refs, fmt.Sprintf(`{"@id":"https://a.example/#t%d"}`, i))
	}
	many := `{"@context":"https://schema.org","@graph":[{"@id":"https://a.example/p/#webpage","@type":"WebPage","url":"https://a.example/p/","about":[` +
		strings.Join(refs, ",") + `]},` + strings.Join(nodes, ",") + `]}`
	fires := []audit.Issue{
		{Path: "$", Severity: audit.Warning, Message: "2 top-level nodes are referenced from other nodes (Piano, Truck): " + looseTail},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"nodes the page points at", loosePage, audittest.Input{}, fires},
		{"a noindex page", loosePage, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"with a keyword and a page", loosePage, audittest.Input{PageText: "Moving a piano.", Keyword: "piano"}, fires},
		{"more than eight", many, audittest.Input{}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "10 top-level nodes are referenced from other nodes (Thing 0, Thing 1, Thing 2, Thing 3, Thing 4, Thing 5, Thing 6, Thing 7, and 2 more): " + looseTail},
		}},
		{"nested where first referenced", keywordPage, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.LooseNodes{}, tc.in, tc.want)
		})
	}
}
