package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestRelatedProducts(t *testing.T) {
	const body = `<section id="also"><h2>You may also like</h2>
<div class="card"><a href="/p/gold-ring/">Gold ring</a><span>$120.00</span></div>
<div class="card"><a href="/p/silver-ring/">Silver ring</a><span>$90.00</span></div></section>`
	listed := graph(`{"@type":"Product","name":"Tea set","isRelatedTo":[{"@type":"Product","name":"Gold ring","url":"https://x.test/p/gold-ring/"},{"@type":"Product","name":"Silver ring","url":"https://x.test/p/silver-ring/"}]}`)
	run(t, content.RelatedProducts{}, append([]tc{
		{"products the Product does not name", at(webPage, body), warn("html:#also", `the page's "You may also like" block lists products (Gold ring, Silver ring) that the Product's isRelatedTo and isSimilarTo do not name: add each as a Product (name, url, image, offers)`)},
		{"products the draft names", at(listed, body), nil},
		// a block of links without prices is the related-links check's
		{"links without prices", at(webPage, `<section id="also"><h2>You may also like</h2><a href="/p/gold-ring/">Gold ring</a> <a href="/p/silver-ring/">Silver ring</a></section>`), nil},
		textOnly(webPage, "You may also like Gold ring $120.00 Silver ring $90.00"),
	}, hidden(webPage, body)...))
}
