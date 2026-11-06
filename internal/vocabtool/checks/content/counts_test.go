package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestCounts(t *testing.T) {
	const body = `<h2>Four ways to say thanks</h2><ol><li>Bow</li><li>Smile</li><li>Toast</li><li>Hug</li></ol>`
	run(t, content.Counts{}, append([]tc{
		{"a heading that counts a list", at(webPage, body), warn("html:#main > h2", `the heading "Four ways to say thanks" counts 4 items: add an ItemList of them (numberOfItems 4, a ListItem each) as the mainEntity of the section's WebPageElement in the WebPage's hasPart, or a HowTo with 4 steps in the hasPart`)},
		{"a digit", at(webPage, `<h2>3 ideas</h2><ul><li>One</li><li>Two</li><li>Three</li></ul>`), warn("html:#main > h2", `the heading "3 ideas" counts 3 items: add an ItemList of them (numberOfItems 3, a ListItem each) as the mainEntity of the section's WebPageElement in the WebPage's hasPart, or a HowTo with 3 steps in the hasPart`)},
		{"an ItemList of that many", at(graph(`{"@type":"ItemList","name":"Thanks","numberOfItems":4}`), body), nil},
		{"a HowTo of that many steps", at(graph(`{"@type":"HowTo","name":"Thanks","step":[{"@type":"HowToStep","name":"a"},{"@type":"HowToStep","name":"b"},{"@type":"HowToStep","name":"c"},{"@type":"HowToStep","name":"d"}]}`), body), nil},
		{"a heading that counts minutes", at(webPage, `<h2>Plan 30 minutes</h2><ul><li>One</li><li>Two</li></ul>`), nil},
		{"a section that holds another count", at(webPage, `<h2>Four ways to say thanks</h2><p>Just say it.</p>`), nil},
		{"a heading that is a link", at(webPage, `<h2><a href="/ideas/">16 ideas</a></h2>`), nil},
		{"a product card's heading", contactPage(at(webPage, `<h2>You may also like</h2><div class="card"><a href="/p/gold/">Gold</a><h3>Three tiny rings</h3><ul><li>a</li><li>b</li><li>c</li></ul><span>$120.00</span></div><div class="card"><a href="/p/silver/">Silver</a><span>$90.00</span></div>`)), nil},
		textOnly(webPage, "Four ways to say thanks"),
	}, hidden(webPage, body)...))
}
