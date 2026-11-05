package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestQuotes(t *testing.T) {
	const tail = ` and no node carries it: a customer's opinion of a business, product or service is a Review of it (reviewBody, author, itemReviewed); another quotation is a Quotation (text, creator)`
	const block = `<figure><blockquote>“She made our day calm and fun.”</blockquote><figcaption><cite>@mei</cite></figcaption></figure>`
	review := graph(`{"@type":"Review","reviewBody":"She made our day calm and fun.","author":{"@type":"Person","name":"@mei"},"itemReviewed":{"@type":"Organization","name":"Ann"}}`)
	run(t, content.Quotes{}, append([]tc{
		{"a blockquote no node carries", at(webPage, block), warn("html:#main > figure > blockquote", `the page quotes "“She made our day calm and fun.”" (@mei)`+tail)},
		{"a pull quote no node carries", at(webPage, `<div class="quote"><cite>@mei</cite><p>“She made our day calm.”</p></div>`), warn("html:#main > div > p", `the page quotes "She made our day calm." (@mei)`+tail)},
		{"a quote its Review carries", at(review, block), nil},
		{"a short quote its Review is", at(graph(`{"@type":"Review","reviewBody":"Great night","author":{"@type":"Person","name":"@kim"},"itemReviewed":{"@type":"Organization","name":"Ann"}}`), `<figure><blockquote>“Great night”</blockquote><figcaption><cite>@kim</cite></figcaption></figure>`), nil},
		{"a quote the text names a speaker of", at(webPage, `<p>“Everyone needs a good chocolate chip cookie recipe,” says culinary producer Nicole McLaughlin (a.k.a. NicoleMcMom ). Here are a few tips.</p>`), warn("html:#main > p", `the page quotes "Everyone needs a good chocolate chip cookie recipe," (culinary producer Nicole McLaughlin)`+tail)},
		{"a quote from no one", at(webPage, `<p>“A line in quotes, from no one.”</p>`), nil},
		{"a quote in the middle of a sentence", at(webPage, `<p>He said “hi” to the room.</p>`), nil},
		textOnly(webPage, "She made our day calm and fun."),
	}, hidden(webPage, block)...))
}
