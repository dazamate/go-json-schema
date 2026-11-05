package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestBusinessInMentions(t *testing.T) {
	fires := []audit.Issue{
		{Path: "$.@graph[2].mentions[0]", Severity: audit.Warning, Message: "U-Haul is another business in mentions: its node ties the page to that business. Keep it only if it is a partner or the page's subject, never a competitor"},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		// an agency, a publisher and a transit district are not businesses
		{"a business in mentions", orgPage, audittest.Input{}, fires},
		{"a noindex page", orgPage, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"the keyword names it", orgPage, audittest.Input{Keyword: "U-Haul truck rental cost"}, nil},
		{"the site's own business", `{"@context":"https://schema.org","@graph":[
			{"@id":"https://a.example/#org","@type":"Organization","name":"Lift My Boxes","additionalType":["https://www.wikidata.org/wiki/Q4830453"]},
			{"@id":"https://a.example/#website","@type":"WebSite","publisher":{"@id":"https://a.example/#org"}},
			{"@type":"WebPage","url":"https://a.example/p/","mentions":[{"@id":"https://a.example/#org"}]}]}`, audittest.Input{}, nil},
		{"no organization in mentions", `{"@type":"WebPage","mentions":[{"@type":"Thing","name":"Piano"}]}`, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.BusinessInMentions{}, tc.in, tc.want)
		})
	}
}
