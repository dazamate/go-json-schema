package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestPublisherLink(t *testing.T) {
	unlinked := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebSite","url":"https://movers.test/","name":"Movers Guide"},
		{"@type":"Organization","name":"Movers Guide","url":"https://movers.test/"}]}`
	linked := published(`"url":"https://movers.test/"`)
	held := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebSite","url":"https://movers.test/","name":"Movers Guide"},
		{"@type":"Organization","@id":"https://movers.test/#org","name":"Movers Guide","url":"https://movers.test/","copyrightHolder":{"@id":"https://movers.test/#org"}}]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"an Organization no WebSite names", audittest.Input{Draft: unlinked, PageHTML: movingGuide, PageURL: movingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `no node publishes the page: set the Organization "Movers Guide" as the WebSite's publisher (and the copyrightHolder), by its @id`},
		}},
		{"a shop's page with an Organization", audittest.Input{Draft: unlinked, PageHTML: audittest.Selling(movingGuide), PageURL: movingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `no node publishes the page: set the Organization "Movers Guide" as the WebSite's publisher (and the copyrightHolder), by its @id`},
		}},
		{"a publisher", audittest.Input{Draft: linked, PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a copyrightHolder", audittest.Input{Draft: held, PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"no Organization", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a noindex page", audittest.Input{Draft: unlinked, PageHTML: audittest.Noindex(movingGuide), PageURL: movingURL}, nil},
		{"the page's text alone", audittest.Input{Draft: unlinked, PageText: "What movers cost. Movers charge by the hour.", PageURL: movingURL}, nil},
		{"no page", audittest.Input{Draft: unlinked, PageURL: movingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.PublisherLink{}, tc.in, tc.want) })
	}
}
