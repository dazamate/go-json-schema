package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestBusinessInAbout(t *testing.T) {
	fires := []audit.Issue{
		{Path: "$.@graph[2].about[0]", Severity: audit.Warning, Message: "PODS (company) is another business in about: the page is not about it, and its node (sameAs, Knowledge Graph ID) ties the page to that business. Drop it if it competes with the site; otherwise move it to mentions"},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"a business in about", orgPage, audittest.Input{}, fires},
		{"a noindex page", orgPage, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"a page with a keyword", orgPage, audittest.Input{Keyword: "moving company", PageText: "Moving."}, fires},
		// the keyword names the business: it is the page's subject
		{"the keyword names it", orgPage, audittest.Input{Keyword: "PODS storage cost"}, nil},
		{"the site's own business", aboutOwn, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.BusinessInAbout{}, tc.in, tc.want)
		})
	}
}

// orgPage names a business and an agency in about, and a business, an agency,
// a publisher and a transit district in mentions.
const orgPage = `{"@context":"https://schema.org","@graph":[
 {"@id":"https://a.example/#org","@type":"OnlineBusiness","name":"Lift My Boxes"},
 {"@id":"https://a.example/#website","@type":"WebSite","publisher":{"@id":"https://a.example/#org"}},
 {"@id":"https://a.example/p/#webpage","@type":"WebPage","url":"https://a.example/p/",
  "about":[
   {"@type":"Organization","name":"PODS (company)","additionalType":["https://www.wikidata.org/wiki/Q4830453"]},
   {"@type":"GovernmentOrganization","name":"United States Postal Service"}],
  "mentions":[
   {"@type":"Organization","name":"U-Haul","additionalType":["https://www.wikidata.org/wiki/Q4830453"]},
   {"@type":"GovernmentOrganization","name":"Federal Motor Carrier Safety Administration"},
   {"@type":"NewsMediaOrganization","name":"CNET"},
   {"@type":"Organization","name":"Los Angeles Metro","additionalType":["https://www.wikidata.org/wiki/Q7835189"]}]}]}`

// aboutOwn has the site's own business in the page's about.
const aboutOwn = `{"@context":"https://schema.org","@graph":[
 {"@id":"https://a.example/#org","@type":"OnlineBusiness","name":"Lift My Boxes"},
 {"@id":"https://a.example/#website","@type":"WebSite","publisher":{"@id":"https://a.example/#org"}},
 {"@id":"https://a.example/p/#webpage","@type":"WebPage","url":"https://a.example/p/","about":[{"@id":"https://a.example/#org"}]}]}`
