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
		{"a star picker of a review form", at(webPage, `<p>Your rating <a>1 of 5 stars</a> <a>2 of 5 stars</a> <a>3 of 5 stars</a> <a>4 of 5 stars</a> <a>5 of 5 stars</a></p>`), nil},
		{"a histogram of reviews by stars", at(webPage, `<ul><li>5 star 15385</li><li>4 star 2359</li><li>3 star 720</li><li>2 star 423</li><li>1 star 446</li></ul>`), nil},
		{"a histogram in attributes", at(webPage, `<a aria-label="5 Stars - 5 Reviews"></a><a aria-label="4 Stars - 3 Reviews"></a><a aria-label="3 Stars - 1 Reviews"></a>`), nil},
		{"a filter list of stars", at(webPage, `<p>By rating 5 star 4 star 3 star 2 star 1 star By type With photo</p>`), nil},
		{"a fraction before the stars", at(webPage, `<p>Superb. I give 4 1/2 stars to the original recipe</p>`), nil},
		{"a product's one rating beside its review count", at(webPage, `<p>5 stars 12 reviews</p>`), warn("html:body", strings.Replace(msg, "(4.8)", "(5)", 1))},
		{"an AggregateRating with that value", at(rated, body), nil},
		{"a gallery's counter", at(webPage, `<p>Photo 1 / 10</p>`), nil},
		{"no rating", at(webPage, `<p>Tea</p>`), nil},
	}, hidden(webPage, body)...))
}
