package content_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestRatings(t *testing.T) {
	const body = `<p>Rated 4.8 out of 5 by couples.</p>`
	const msg = "the page shows ratings (4.8) that no Rating or AggregateRating carries as ratingValue: mark each up on the Review or item it rates (ratingValue, bestRating, and ratingCount for an aggregate)"
	rated := graph(`{"@type":"Organization","name":"Ann","aggregateRating":{"@type":"AggregateRating","ratingValue":4.8,"bestRating":5,"ratingCount":12}}`)
	run(t, content.Ratings{}, append([]tc{
		{"a rating no node carries", at(webPage, body), warn("html:body", msg)},
		{"a rating in an aria-label", at(webPage, `<span aria-label="4.8 out of 5 stars"></span>`), warn("html:body", msg)},
		{"the text alone", audittest.Input{Draft: webPage, PageText: "Rated 4.8 out of 5 by couples.", PageURL: pageURL}, warn("html:body", msg)},
		{"two decimals", at(webPage, `<p>Rated 4.75/5 stars</p>`), warn("html:body", strings.Replace(msg, "(4.8)", "(4.75)", 1))},
		{"an AggregateRating with that value", at(rated, body), nil},
		{"a gallery's counter", at(webPage, `<p>Photo 1 / 10</p>`), nil},
		{"no rating", at(webPage, `<p>Tea</p>`), nil},
	}, hidden(webPage, body)...))
}
