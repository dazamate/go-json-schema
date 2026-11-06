package webpage

import (
	"strings"
	"testing"
)

// TestHTMLTextSkipsSVGNames checks that the title and desc of an icon in a
// link or button, which name the icon for a screen reader and show as no
// text, are no visible text (taylorstitch.com's menu links read "New &
// Featured Nav chevron"), while a chart's text labels are, and a graphic
// outside a control keeps its title: royalcanadiancollective.com's row of
// card logos says how to pay.
func TestHTMLTextSkipsSVGNames(t *testing.T) {
	doc := parse(t, `<html><body><ul class="main-nav--menu"><li class="main-nav--link" id="main-nav-link-new-featured" data-link-id="new-featured">
		<a href="/collections/mens-new-arrivals" class="">
		New &amp; Featured <span><svg class="icon icon-nav-chevron" focusable="false" aria-label="Nav chevron"><title>Nav chevron</title><desc>An arrow</desc><use xlink:href="#icon-nav-chevron"></use>
		</svg></span></a></li></ul>
		<button type="button" data-submenu-close="" aria-label="Close main navigation"><svg class="icon icon-nav-close" focusable="false" aria-label="Close main navigation"><title>Close main navigation</title><use xlink:href="#icon-nav-close"></use></svg></button>
		<figure><svg viewBox="0 0 10 10"><text x="1" y="1">62% of bloggers</text></svg><figcaption>Bloggers by year</figcaption></figure>
		<ul><li><svg class="oddit-footer__payment-icon" xmlns="http://www.w3.org/2000/svg" role="img" viewBox="0 0 38 24" width="38" height="24" fill="none" aria-labelledby="pi-visa"><title id="pi-visa">Visa</title><rect x=".5" y=".5" width="37" height="23" rx="3"></rect></svg></li></ul></body></html>`)
	got := HTMLText(doc)
	if want := "New & Featured 62% of bloggers Bloggers by year Visa"; got != want {
		t.Errorf("HTMLText = %q, want %q", got, want)
	}
	if lines := strings.Join(MainLines(doc), " | "); strings.Contains(lines, "chevron") || !strings.Contains(lines, "62% of bloggers") {
		t.Errorf("MainLines = %q", lines)
	}
}

// TestCartControlReadsSVGName checks that a cart button whose only name is its
// icon's svg title is still the cart (taylorstitch.com "Open Cart Menu").
func TestCartControlReadsSVGName(t *testing.T) {
	doc := parse(t, `<html><body><a href="/cart" id="open-right-cart" class="open-right-drawer open-cart" data-section-to-show="#right-drawer-cart" data-opendrawer="right"><svg class="icon icon-nav-cart" focusable="false"><title>Open Cart Menu</title><use xlink:href="#icon-nav-cart"></use>
		</svg><sup class="cart-count visually-hidden" data-cart-count="">0</sup><span class="visually-hidden" data-cart-count-label=""> items</span></a>
		<a href="/collections/mens-new-arrivals">New &amp; Featured <span><svg class="icon icon-nav-chevron" focusable="false"><title>Nav chevron</title></svg></span></a></body></html>`)
	bs := QueryAll(doc, "a")
	if !CartControl(bs[0]) {
		t.Error("the cart button named by its svg title is no cart control")
	}
	if CartControl(bs[1]) {
		t.Error("a menu button is a cart control")
	}
}

// TestCopyrightLine checks copyright lines as sites word them: the holder
// before the year (loveitlandscaping.com.au, sublimecandles.co.uk,
// oliverbonas.com), an "@" after the mark (tutorfinder.com.au), a range, and
// the year before the mark; business.gov.au's menu of IP rights is no copyright
// line.
func TestCopyrightLine(t *testing.T) {
	for text, want := range map[string][2]string{
		"© Love It Landscaping 2023. All Rights Reseverd":                                  {"", "2023"},
		"Terms & conditions © Copyright Sublime Candles 2026":                              {"", "2026"},
		"Read our 566151 reviews © Oliver Bonas 2026 Privacy Policy":                       {"", "2026"},
		"Copyright @ 2026 TutorFinder.com.au English Tutors":                               {"", "2026"},
		"©2009–2026 Moorehead Dentistry":                                                   {"2009", "2026"},
		"Copyright 2026":                                                                   {"", "2026"},
		"a trade mark Copyright Designs Circuit layouts Plant breeder's rights What to do": {"", ""},
	} {
		got := [2]string{}
		if m := CopyrightLine.FindStringSubmatch(text); m != nil {
			got = [2]string{m[1], m[2]}
		}
		if got != want {
			t.Errorf("CopyrightLine(%q) = %q, want %q", text, got, want)
		}
	}
	if m := CopyrightYearFirst.FindStringSubmatch("2026 © Acme Pty Ltd"); m == nil || m[1] != "2026" {
		t.Errorf("CopyrightYearFirst = %q", m)
	}
}

// TestMainSkipsSiteFooterHolderFirst checks that a footer inside main whose
// copyright line names the holder before the year is the site's footer, as
// one with "© 2026" is.
// TestPolicyText checks that the overlays named for a shop's shipping,
// returns or guarantee give their text (taylorstitch.com's dialog#shipping-
// dialog, a "Shipping & Returns" drawer), and a cart, wishlist or help one
// does not.
func TestPolicyText(t *testing.T) {
	doc := parse(t, `<html><body><main><h1>Shirt</h1></main>
		<dialog id="shipping-dialog"><p>Return an item for a full refund within 21 days.</p></dialog>
		<div class="drawer" role="dialog"><h2>Shipping &amp; Returns</h2><p>Exchanges are free.</p></div>
		<div class="oddit-cart-drawer"><p>Your cart $299.00</p></div>
		<div role="dialog" aria-label="My wishlists"><p>View store credit.</p></div>
		<div class="modal" role="dialog"><p>Earn 1 point for every $1 spent.</p></div></body></html>`)
	got := New(doc, HTMLText(doc), "https://shop.test/products/shirt").PolicyText()
	for _, want := range []string{"full refund", "Exchanges are free"} {
		if !strings.Contains(got, want) {
			t.Errorf("PolicyText() = %q, want %q in it", got, want)
		}
	}
	for _, not := range []string{"$299.00", "store credit", "Earn 1 point", "Shirt"} {
		if strings.Contains(got, not) {
			t.Errorf("PolicyText() = %q, want no %q", got, not)
		}
	}
}

func TestMainSkipsSiteFooterHolderFirst(t *testing.T) {
	doc := parse(t, `<html><body><main><h1>Returns and Refunds</h1><p>You can return most items to us within 30 days.</p>
		<footer><p>Read our 566151 reviews</p><p>© Oliver Bonas 2026</p><a href="/privacy">Privacy Policy</a></footer></main></body></html>`)
	p := New(doc, HTMLText(doc), "https://www.oliverbonas.com/returns-and-refunds")
	if strings.Contains(p.Main, "566151 reviews") || !strings.Contains(p.Main, "within 30 days") {
		t.Errorf("Main = %q", p.Main)
	}
}

// TestChromeByClass checks that a div the site names its header, footer or
// menu by a whole class or id token, or a cookie consent block, is chrome
// like the elements: officeworks.com.au's div#footer, jamieoliver.com's
// div.header, loveitlandscaping.com.au's div.menu, orbitmedia.com's
// Cookiebot dialog, a Shopify theme's footer group. A token that only holds
// the word (vicisouthbank.com.au's price list headers, reliancedigital.in's
// product description header, tutorfinder.com.au's rates card header), a
// page header, and a menu inside the main copy (a restaurant's) are no
// chrome.
func TestChromeByClass(t *testing.T) {
	doc := parse(t, `<html><body>
		<div class="header astro-hhcwb6lh"><div class="header__bar-auth--content"><a id="c1" href="/login">Sign In or Register</a></div></div>
		<div class="menu"><span class="close-menu icon-cross2 right-boxed"></span><div class="menu-lang right-boxed lang_sw"><p><a id="c2" title="English" href="#">Eng</a></p></div></div>
		<div id="CybotCookiebotDialog" name="CybotCookiebotDialog" role="region" class="CybotEdge CybotMultilevel"><a id="c3" href="https://www.cookiebot.com">Cookiebot</a></div>
		<div id="shopify-section-sections--19701750038726__footer" class="shopify-section shopify-section-group-group-footer"><a id="c4" href="/pages/contact">Contact us</a></div>
		<div id="shopify-section-sections--2__footer" class="shopify-section shopify-section-group-footer-group"><a id="c5" href="/pages/contact">Contact us</a></div>
		<main>
		<div class="opus-page"><div class="page-header-wrapper"><div class="page-header outside text-center"><h1 id="k1">Opus Vancouver Boutique Hotel in Yaletown</h1></div></div></div>
		<div class="elementor-price-list-header"><span id="k2" class="elementor-price-list-title">Acai Activate</span><span class="elementor-price-list-price">11</span></div>
		<div class="product-description-header"><a id="k3" href="#info">Product Info</a></div>
		<div class="tf-card-header"><h2 id="k4">Average Hourly Rate Charged by Tutors</h2></div>
		<div class="menu"><h2 id="k5">Breakfast</h2><p>Eggs on toast $18</p></div>
		<p class="comment-form-cookies-consent"><label id="k6">Save my name, email, and website in this browser for the next time I comment.</label></p>
		</main>
		<div id="footer"><!-- BEGIN Footer.jsp --><a id="c6" href="/shop/officeworks/newsletter">Want to keep in touch?</a></div>
		</body></html>`)
	for _, id := range []string{"c1", "c2", "c3", "c4", "c5", "c6"} {
		if n := QueryAll(doc, "#"+id); !Chrome(n[0]) {
			t.Errorf("#%s is no chrome", id)
		}
	}
	for _, id := range []string{"k1", "k2", "k3", "k4", "k5", "k6"} {
		if n := QueryAll(doc, "#"+id); Chrome(n[0]) {
			t.Errorf("#%s is chrome", id)
		}
	}
}
