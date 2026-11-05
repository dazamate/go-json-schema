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
func TestMainSkipsSiteFooterHolderFirst(t *testing.T) {
	doc := parse(t, `<html><body><main><h1>Returns and Refunds</h1><p>You can return most items to us within 30 days.</p>
		<footer><p>Read our 566151 reviews</p><p>© Oliver Bonas 2026</p><a href="/privacy">Privacy Policy</a></footer></main></body></html>`)
	p := New(doc, HTMLText(doc), "https://www.oliverbonas.com/returns-and-refunds")
	if strings.Contains(p.Main, "566151 reviews") || !strings.Contains(p.Main, "within 30 days") {
		t.Errorf("Main = %q", p.Main)
	}
}
