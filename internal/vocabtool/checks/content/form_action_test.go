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
		// forms that send no message to the business: the page's own widgets
		{"a country switcher", at(webPage, `<form action="/localization" method="post" id="localization_form"><input type="hidden" name="country_code" value="GB"><button type="submit">United Kingdom (GBP)</button></form>`), nil},
		{"a login form", at(webPage, `<form action="/account/login" method="post" id="customer_login"><input type="email" name="customer[email]"><input type="password" name="customer[password]"><input type="submit" value="Sign in"></form>`), nil},
		{"a password reset form", at(webPage, `<form action="/account/recover" method="post" id="sidebar-recover-password"><p>Reset Password</p><input type="email" name="email"><input type="submit" value="Submit"></form>`), nil},
		{"a sort form of choices", at(webPage, `<form id="CollectionFiltersForm"><input type="radio" name="sort_by" value="best-selling"><label>Best selling</label><input type="radio" name="sort_by" value="title-ascending"><label>Alphabetically, A-Z</label></form>`), nil},
		{"a help picker of choices", at(webPage, `<form action="#"><p>What can we help you with today?</p><input type="radio" name="q" value="mortgage"><input type="radio" name="q" value="savings"></form>`), nil},
		{"a poll of choices", at(webPage, `<form action="/formbuilder?sc_site=x" method="post" id="fxb_1"><input type="hidden" name="fxb.1" value="0"><p>Was this page helpful?</p><input type="radio" id="btnlist-yes" name="fxb.1" value="Yes"><input type="radio" id="btnlist-no" name="fxb.1" value="No"></form>`), nil},
		{"a site search box of a text field", at(webPage, `<div id="hd"><form class="hd-Banner_SearchForm" action="/pages/search-results"><input type="text" id="Search" name="q" placeholder="What are you looking for?"></form></div>`), nil},
		{"a print button", at(webPage, `<form id="print" method="GET" action="/recipe/10813/?print"><button>Print</button><input type="hidden" name="print"></form>`), nil},
		{"a button that goes to another page", at(webPage, `<div id="bar"><form action="https://x.test/register"><input type="submit" name="Action" value="Place Tutor Ad"></form></div>`), nil},
		{"a calculator the page's script runs", at(webPage, `<form><p>My height is <input type="number" name="ft"> feet</p><button type="button">What's My Size?</button></form>`), nil},
		{"a form around the whole page", at(webPage, `<form method="post" action="./12434" id="frmDefault"><input type="hidden" name="__VIEWSTATE" value="x"><nav><a href="/">Home</a></nav><input type="number" name="price"><footer>Contact</footer></form>`), nil},
		{"a site search with a search page address", at(webPage, `<div id="tf"><form action="https://x.test/tutors/search.php" method="post"><input type="text" name="subject_text"><input type="checkbox" name="online"><input type="submit" value="Search"></form></div>`), nil},
		{"a delivery zip lookup", at(webPage, `<div id="variants"><form class="form--edd" method="post"><input type="text" id="zipcode" name="zipcode" placeholder="Enter your ZIP code"><button type="submit"></button></form></div>`), nil},
		{"an order lookup", at(webPage, `<div id="modal"><form action="/information/track-your-order" method="get"><input type="text" name="orderId" placeholder="Enter Order Number Here"><button type="submit">Track your order</button></form></div>`), nil},
		{"a translate widget's vote form", at(webPage, `<div id="pane"><form id="goog-gt-votingForm" action="//translate.googleapis.com/translate_voting?client=te" method="post"><input type="text" name="sl"><input type="text" name="tl"><input type="text" name="query"></form></div>`), nil},
		{"a cart drawer's checkout", at(webPage, `<div id="drawer"><form class="psh-CartFooter_Buttons" action="/cart" method="POST"><button type="submit" name="checkout" id="checkout">Checkout Securely</button><a href="/cart">View cart</a></form></div>`), nil},
		{"a cart drawer's checkout with a note field", at(webPage, `<div id="drawer"><form action="/cart" method="POST"><textarea name="note"></textarea><button type="submit" name="checkout">Checkout</button></form></div>`), nil},
		// the other form checks have these
		{"a pre-order form", at(webPage, `<form action="/pre-order"><button>Pre-order now</button></form>`), nil},
		{"an add-to-bag form", at(webPage, `<form action="/cart/add"><button>Add to bag</button></form>`), nil},
		{"a newsletter form", at(webPage, `<section id="join"><p>Subscribe to our newsletter</p><form><input type="email" name="email"></form></section>`), nil},
	}, formGates(body)...))
}
