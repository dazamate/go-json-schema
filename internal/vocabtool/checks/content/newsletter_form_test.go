package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestNewsletterForm(t *testing.T) {
	body := `<section id="join"><p>Subscribe to our newsletter</p><form action="/subscribe" method="post"><input type="email" name="email"></form></section>`
	run(t, content.NewsletterForm{}, append([]tc{
		{"a newsletter form", at(webPage, body), warn("html:#join > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #join")},
		{"a signup button worded as a sign me up", at(webPage, `<div id="sticky"><form id="gform_2" class="newsletter-signup-form" method="post" action="/blog/stats/"><input type="text" name="input_5"><input type="email" name="input_1"><button type="submit">Sign me up</button></form></div>`), warn("html:#gform_2", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #gform_2")},
		{"a form of the footer newsletter section", at(webPage, `<section id="shopify-section-footer-newsletter"><form><input type="email" name="email" placeholder="Email"><input type="submit" value="Email"></form></section>`), warn("html:#shopify-section-footer-newsletter > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #shopify-section-footer-newsletter")},
		{"a form whose email field is the newsletter's", at(webPage, `<div id="foot"><p>Occasional and only ask.</p><form class="w-full"><label for="newsletter-email">Email address</label><input id="newsletter-email" type="email" name="email"><button type="submit">Send me the occasional note</button></form></div>`), warn("html:#foot > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #foot")},
		{"an account sign up with a password", at(webPage, `<section id="create_customer"><p>Already an account? Log In. Sign up</p><form action="/account" method="post"><input type="email" name="customer[email]"><input type="password" name="customer[password]"></form></section>`), nil},
		{"a form with no email field", at(webPage, `<section id="join"><p>Subscribe to our newsletter</p><form action="/subscribe"><input name="name"></form></section>`), nil},
	}, formGates(body)...))
}
