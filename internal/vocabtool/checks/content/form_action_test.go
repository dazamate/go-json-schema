package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestFormAction(t *testing.T) {
	const msg = "the page's form has no Action whose target points at it: add the Action it performs (AskAction, QuoteAction, ReserveAction, SubscribeAction and the like) "
	body := `<section id="enquire"><form method="POST"><input name="name"></form></section>`
	run(t, content.FormAction{}, append([]tc{
		{"a form in a section with an id", at(webPage, body), warn("html:#enquire > form", msg+"with an EntryPoint target whose urlTemplate is the page URL plus #enquire")},
		{"a form with an id of its own", at(webPage, `<form id="contact" method="POST"><input name="name"></form>`), warn("html:#contact", msg+"with an EntryPoint target whose urlTemplate is the page URL plus #contact")},
		{"a form with no id around it", bare(`<form method="POST"><input name="name"></form>`), warn("html:body > form", msg+"give the form or its section an id and point the target's urlTemplate at it")},
		// the other form checks have these
		{"a pre-order form", at(webPage, `<form action="/pre-order"><button>Pre-order now</button></form>`), nil},
		{"an add-to-bag form", at(webPage, `<form action="/cart/add"><button>Add to bag</button></form>`), nil},
		{"a newsletter form", at(webPage, `<section id="join"><p>Subscribe to our newsletter</p><form><input type="email" name="email"></form></section>`), nil},
	}, formGates(body)...))
}
