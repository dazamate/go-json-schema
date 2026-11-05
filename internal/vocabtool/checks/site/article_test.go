package site_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestArticle(t *testing.T) {
	terms := strings.Replace(movingGuide, "<h1>What movers cost</h1>", "<h1>Terms and conditions</h1>", 1)
	cards := `<!doctype html><html><body><main><h1>Compare</h1><article><h3>Movers</h3><p>A crew lifts and drives.</p></article><article><h3>Truck</h3><p>You drive.</p></article></main></body></html>`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a long article", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: movingURL}, []audit.Issue{
			{Path: "html:body > main > article", Severity: audit.Warning, Message: "the page's main copy is an article (173 words, 2 sections) but no node is an Article: add one (headline from the h1, articleSection, wordCount, author, datePublished, dateModified, publisher, isPartOf and mainEntityOfPage the WebPage), and give the WebPage a potentialAction ReadAction targeting its URL"},
		}},
		{"an Article in the draft", audittest.Input{Draft: withNodes(`{"@type":"Article","headline":"What movers cost"}`), PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a terms page by its heading", audittest.Input{Draft: bareDraft, PageHTML: terms, PageURL: "https://movers.test/rules/"}, nil},
		{"a terms page by its address", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: "https://movers.test/terms-and-conditions/"}, nil},
		{"cards", audittest.Input{Draft: bareDraft, PageHTML: cards, PageURL: "https://movers.test/compare/"}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(movingGuide), PageURL: movingURL}, nil},
		{"the page's text alone", audittest.Input{Draft: bareDraft, PageText: "What movers cost. Movers charge by the hour.", PageURL: movingURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: movingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.Article{}, tc.in, tc.want) })
	}
}
