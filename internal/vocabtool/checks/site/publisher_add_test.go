package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestPublisherAdd(t *testing.T) {
	add := func(name string) string {
		return `no node publishes the page: add the business as an Organization (or the subtype that fits it, or the Person who runs the site) with ` + name + `, url, logo, and the page's telephone and email, and set it as the WebSite's publisher`
	}
	unlinked := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebSite","url":"https://movers.test/","name":"Movers Guide"},
		{"@type":"Organization","name":"Movers Guide","url":"https://movers.test/"}]}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"the © line's holder", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: movingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: add(`name "Movers Guide" from the © line`)},
		}},
		{"no © line", audittest.Input{Draft: bareDraft, PageHTML: `<!doctype html><html><body><main><h1>What movers cost</h1><p>Movers charge by the hour.</p></main></body></html>`, PageURL: movingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: add("name")},
		}},
		{"a publisher", audittest.Input{Draft: published(`"url":"https://movers.test/"`), PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"an Organization to link", audittest.Input{Draft: unlinked, PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a shop's page with no Organization", audittest.Input{Draft: bareDraft, PageHTML: audittest.Selling(movingGuide), PageURL: movingURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(movingGuide), PageURL: movingURL}, nil},
		{"the page's text alone", audittest.Input{Draft: bareDraft, PageText: "What movers cost. © 2026 Movers Guide", PageURL: movingURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: movingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.PublisherAdd{}, tc.in, tc.want) })
	}
}
