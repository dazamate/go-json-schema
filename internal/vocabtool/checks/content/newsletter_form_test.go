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
		// a sign-up's other fields, and the textareas a captcha or a spam trap hides in it
		{"a sign-up with a phone field for texts", at(webPage, `<section id="signup"><p>Sign up for emails or texts</p><form><input type="email" name="email"><input type="tel" name="phone"><button>Subscribe</button></form></section>`), warn("html:#signup > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #signup")},
		{"a sign-up with dropdowns", at(webPage, `<section id="news"><p>Get the latest news</p><form action="/subscribe"><input type="email" name="emailAddress"><select name="gender"><option>Female</option><option>Male</option></select></form></section>`), warn("html:#news > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #news")},
		{"a sign-up with a captcha's hidden textarea", at(webPage, `<section id="news"><p>Sign up</p><form><input type="email" name="email"><textarea name="g-recaptcha-response" style="width: 250px; display: none;"></textarea></form></section>`), warn("html:#news > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #news")},
		{"a sign-up with a hidden spam trap textarea", at(webPage, `<section id="news"><p>Sign me up</p><form><input type="email" name="input_1"><p style="display: none !important;"><textarea name="ak_hp_textarea"></textarea></p></form></section>`), warn("html:#news > form", "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) with an EntryPoint target whose urlTemplate is the page URL plus #news")},
		// a message box makes a form an enquiry or a comment, whatever its words say
		{"an enquiry form beside a no mailing list line", at(webPage, enquiry(`<textarea name="details"></textarea>`)), nil},
		{"a comment form with a subscribe to replies dropdown", at(webPage, `<div id="respond"><form action="/wp-comments-post.php" method="post" id="commentform"><textarea name="comment"></textarea><input name="author"><input type="email" name="email"><select name="subscribe-reloaded"><option value="yes">Subscribe to replies</option></select></form></div>`), nil},
	}, formGates(body)...))
}

// enquiry is an enquiry form with the field given, beside mcmo.au's line
// that its details go on no mailing list.
func enquiry(field string) string {
	return `<section id="enquire"><form method="POST" action="https://formsubmit.co/ajax/x@example.com"><input name="name"><input name="email" type="email"><input name="date" type="date"><input name="phone" type="tel">` + field + `<button type="submit">Check your date</button></form><p>Your details are used only to answer you: no mailing list, no sales calls.</p></section>`
}
