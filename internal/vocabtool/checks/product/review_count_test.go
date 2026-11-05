package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestReviewCount(t *testing.T) {
	page := plainPageWith("<p>Rated by 128 reviews.</p>")
	carried := `{"@context":"https://schema.org","@type":"Product","aggregateRating":{"@type":"AggregateRating","ratingValue":4.8,"reviewCount":128}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a review count", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page counts 128 reviews: set the Product's aggregateRating, an AggregateRating (ratingValue, bestRating, reviewCount 128)`},
		}},
		{"the reviewCount is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"no review count", audittest.Input{Draft: noDraft, PageHTML: plainPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Rated by 128 reviews.`, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page counts 128 reviews: set the Product's aggregateRating, an AggregateRating (ratingValue, bestRating, reviewCount 128)`},
		}},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.ReviewCount{}, tc.in, tc.want) })
	}
}
