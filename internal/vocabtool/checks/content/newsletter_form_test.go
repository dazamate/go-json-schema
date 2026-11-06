package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestNewsletterForm(t *testing.T) {
	body := `<section id="join"><p>Subscribe to our newsletter</p><form action="/subscribe" method="post"><input type="email" name="email"></form></section>`
	run(t, content.NewsletterForm{}, append([]tc{
		{"a newsletter form", at(webPage, body), warn("html:#join > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #join")},
		{"a form with no email field", at(webPage, `<section id="join"><p>Subscribe to our newsletter</p><form action="/subscribe"><input name="name"></form></section>`), nil},
	}, formGates(body)...))
}
