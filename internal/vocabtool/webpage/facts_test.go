package webpage

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// parse reads a page the way the checker's callers do.
func parse(t *testing.T, src string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestNewClassifiesProductPage(t *testing.T) {
	doc := parse(t, `<html><head><meta property="og:type" content="product"></head>
		<body><h1>Gold locket</h1><p>$49.00</p>
		<form action="/cart/add"><button>Add to cart</button></form></body></html>`)
	p := New(doc, HTMLText(doc), "https://shop.test/products/gold-locket")
	if p.Kind != KindProduct {
		t.Errorf("Kind = %q, want %q", p.Kind, KindProduct)
	}
	if !p.Sells() {
		t.Error("a page with an add-to-cart form should sell")
	}
	if p.Private != "" {
		t.Errorf("Private = %q on an indexed page", p.Private)
	}
}

func TestNewPrivateNoindex(t *testing.T) {
	doc := parse(t, `<html><head><meta name="robots" content="noindex"></head><body><h1>Thanks</h1></body></html>`)
	p := New(doc, HTMLText(doc), "https://shop.test/thanks")
	if p.Private != "noindex" {
		t.Errorf("Private = %q, want noindex", p.Private)
	}
}

// TestMainSkipsLandmarkRoles checks that a div the page calls its navigation,
// banner or footer is its chrome, like the elements: officeworks.com.au's
// mobile menu, smithpartners.co.nz's header and footer.
func TestMainSkipsLandmarkRoles(t *testing.T) {
	doc := parse(t, `<html><body><div id="header"><div id="mobile-menu" class="ow-mobile-menu ow-drawer ow-drawer--left" role="navigation">
		<ul class="ow-mobile-menu__item"><li> <a href="/shop/officeworks/c/gifting/gifts-by-price/gifts-under--10">Gifts $10 and Under</a> </li></ul></div></div>
		<div id="flex-header" data-sticky="true" role="banner"><a href="/">Smith and Partners</a> Contact Us Today</div>
		<main><h1>Contact us</h1><p>Call us on 13 00 OFFICE.</p></main>
		<div id="flex-footer" role="contentinfo">Lincoln Manor 293 Lincoln Road, Henderson, Auckland 0610</div></body></html>`)
	p := New(doc, HTMLText(doc), "https://www.officeworks.com.au/contact-us")
	for _, s := range []string{"Gifts $10", "Contact Us Today", "Lincoln Manor"} {
		if strings.Contains(p.Main, s) {
			t.Errorf("Main holds the chrome %q: %q", s, p.Main)
		}
	}
	if !strings.Contains(p.Main, "Call us on") {
		t.Errorf("Main = %q", p.Main)
	}
	if a := QueryAll(doc, "#mobile-menu a")[0]; !Chrome(a) {
		t.Error("a link in a navigation landmark should be chrome")
	}
}

// TestAddressInFooter checks that a business's address in its site footer
// is the page's: loveitlandscaping.com.au, vicisouthbank.com.au and
// smithpartners.co.nz give theirs only there.
func TestAddressInFooter(t *testing.T) {
	for src, want := range map[string]string{
		`<html><body><main><h1>Landscaping costs in Melbourne</h1></main><footer><div class="col-contact contact-footer"><h2 class="contact-title contact-top">Melbourne, Australia</h2> <p class="contact-address text-muted"><strong>11 Frederick St, Sunbury, VIC 3429</strong></p></div><ul><li><a href="/">Home</a></li></ul></footer></body></html>`: "11 Frederick St, Sunbury, VIC 3429",
		`<html><body><main><h1>Menu</h1></main><footer><div class="elementor-widget-container"> <p>174 Grey St, South Brisbane 4101</p> </div><p>Trading Open 7 Days</p></footer></body></html>`:                                                                                                                                                           "174 Grey St, South Brisbane 4101",
		`<html><body><main><h1>Property lawyers</h1></main><div id="flex-footer" role="contentinfo"><p>Lincoln Manor</p><p>293 Lincoln Road, Henderson, Auckland 0610</p></div></body></html>`:                                                                                                                                                             "293 Lincoln Road, Henderson, Auckland 0610",
		// jamesplumbingheating.com: the address runs across the footer's lines
		`<html><body><main><h1>Pricing</h1></main><footer><div itemprop="address" itemscope="" itemtype="https://schema.org/PostalAddress"><p><span itemprop="streetAddress">15, Coniston Road</span><br><span itemprop="addressLocality">Twickenham</span> <span itemprop="postalCode">TW2 7HH</span></p></div><p><a href="tel:02080508928">0208 050 8928</a></p></footer></body></html>`: "15, Coniston Road Twickenham TW2 7HH",
	} {
		doc := parse(t, src)
		if got := New(doc, HTMLText(doc), "https://site.test/page").Address(); got != want {
			t.Errorf("Address() = %q, want %q", got, want)
		}
	}
}

// TestAddressEnds checks that a street address stops where the address
// does: at its postcode or ZIP and the country after it, else before the
// words that start new copy, on pages from NZ, AU, the UK and the US; that a
// home for sale in an article and a returns depot are no business's
// address; and that a house number may take a comma ("15, Coniston Road").
func TestAddressEnds(t *testing.T) {
	for _, r := range []struct{ name, src, want string }{
		{"autodiscount.co.nz, no postcode", `<div class="vehicle-description">nal servicing for $300<br><br>Auto Discount<br>10 Doncaster Street, Mngere, Auckland<br>PH0NE O9 39O O3O3<br> </div> <div class="small-12 columns feature-wrap hide-for-medium-only"> <h4>Feature Highlights</h4>`, "10 Doncaster Street, Mngere, Auckland"},
		{"officeworks.com.au, AU postcode", `<p><a href="/shop/officeworks/storelocator">Or find your local Officeworks store</a></p> <div class="ow-contact-info__address"> <p>Officeworks Support Centre:</p> Chadstone Place, 1341 Dandenong Rd, Chadstone VIC 3148</div> <form><label>What are you contacting us about?</label></form>`, "1341 Dandenong Rd, Chadstone VIC 3148"},
		{"dentistryforyou.co.uk, UK postcode", `<p>Harunani and Co Group of Dental Surgeries t/a Dentistry for you has its address at 45 Southend Road, Hockley, United Kingdom, SS5 4PZ and is authorised and regulated by the Financial Conduct Authority.</p>`, "45 Southend Road, Hockley, United Kingdom, SS5 4PZ"},
		{"mooreheaddentistry.com, US ZIP", `<div class="column wysiwyg"> <h3>Blue Ash Dentistry</h3> <p> 10427 Kenwood Road<br> Cincinnati, OH 45242 <br><a style="text-decoration: underline;" href="https://goo.gl/maps/7pFmrok3avE2cSCS7" target="_blank">View Map</a> </p> <h4>Hours</h4> <p>Mon: 8AM - 4PM</p></div>`, "10427 Kenwood Road Cincinnati, OH 45242"},
		{"a unit number", `<p>1521 Walmart Drive #301<br> Lebanon, OH 45036 <br><a href="https://goo.gl/maps/x">View Map</a></p>`, "1521 Walmart Drive #301 Lebanon, OH 45036"},
		{"opushotel.com, CA postal code and country", `<p>322 Davie St<br>Vancouver, BC V6B 5Z6 Canada</p><p>Phone +1 866-642-6787</p>`, "322 Davie St Vancouver, BC V6B 5Z6 Canada"},
		{"jamesplumbingheating.com, a comma after the number", `<div itemprop="address" itemscope="" itemtype="https://schema.org/PostalAddress"><p><span itemprop="streetAddress">15, Coniston Road</span><br><span itemprop="addressLocality">Twickenham</span> <span itemprop="postalCode">TW2 7HH</span></p></div><p><a href="tel:02080508928">0208 050 8928</a></p>`, "15, Coniston Road Twickenham TW2 7HH"},
		{"redfin.com, homes for sale in a blog post", `<h2>Checklist for buying a house</h2><div class="house-listing"><div class="price"><strong>$1,850,000</strong></div><div class="details">5 beds, 3.5 baths, 4228 sq ft</div><div class="address"><a href="https://www.redfin.com/IL/Chicago/4130-N-Kenneth-Ave-60641/home/22650100" target="_blank">4130 N Kenneth Ave, Chicago, IL</a></div></div><div class="house-listing"><div class="price"><strong>$579,900</strong></div><div class="details">3 beds, 2.5 baths, 1692 sq ft</div><div class="address"><a href="https://www.redfin.com/IL/Schaumburg/1425-Concord-Ln-60193/home/13716465" target="_blank">1425 Concord Ln, Schaumburg, IL</a></div></div><div class="house-listing"><div class="price"><strong>$1,799,000</strong></div><div class="details">4 beds, 3 baths, 1936 sq ft</div></div>`, ""},
		{"oliverbonas.com, a returns depot", `<p>Customers in the rest of the world are responsible for any local postage costs associated with returns and orders should be sent back to – Oliver Bonas Returns, Unit 1, Lions Park Avenue, Chessington, KT9 1ST, United Kingdom. Please include a completed returns form.</p>`, ""},
		{"a year is no house number", `<p>In 2026, Ocean Drive Road works closed the bridge.</p>`, ""},
	} {
		doc := parse(t, "<html><body><main>"+r.src+"</main></body></html>")
		if got := New(doc, HTMLText(doc), "https://site.test/page").Address(); got != r.want {
			t.Errorf("%s: Address() = %q, want %q", r.name, got, r.want)
		}
	}
}

// TestHoursInFooter checks that a business's hours in its site footer are
// the page's: autodiscount.co.nz gives its yard's only there.
func TestHoursInFooter(t *testing.T) {
	doc := parse(t, `<html><body><main><h2>2015 Toyota Aqua</h2><p>$14,450</p></main>
		<footer><h4>Opening hours</h4><p>Monday 9.30am - 5.30pm</p><p>Sunday Closed</p><p>©2018 - 2026 Auto Discount</p></footer></body></html>`)
	if got, want := New(doc, HTMLText(doc), "https://www.autodiscount.co.nz/vehicle/2015-Toyota-Aqua/12434").Hours(), "Monday 9.30am - 5.30pm"; got != want {
		t.Errorf("Hours() = %q, want %q", got, want)
	}
	doc = parse(t, `<html><body><main><h1>Fees</h1><p>Mon: 8AM - 4PM</p></main><footer><p>Sunday Closed</p></footer></body></html>`)
	if got, want := New(doc, HTMLText(doc), "https://www.mooreheaddentistry.com/financial/office-fees/").Hours(), "Mon: 8AM - 4PM"; got != want {
		t.Errorf("Hours() = %q, want %q", got, want)
	}
}

func TestMainSkipsChrome(t *testing.T) {
	doc := parse(t, `<html><body><header><a href="/">Acme Shop</a></header>
		<main><h1>Lockets</h1><p>Hand made.</p></main></body></html>`)
	p := New(doc, HTMLText(doc), "https://shop.test/pages/lockets")
	if strings.Contains(p.Main, "Acme Shop") {
		t.Errorf("Main holds the site header: %q", p.Main)
	}
	if !strings.Contains(p.Main, "Hand made.") || !strings.Contains(p.Text, "Acme Shop") {
		t.Errorf("Main = %q, Text = %q", p.Main, p.Text)
	}
}

// TestMainSkipsOverlays checks that a drawer, modal or dialog the page opens
// over itself is no main copy (royalcanadiancollective.com's closed cart
// drawer with its picks' prices, officeworks.com.au's help modal "Earn 1
// point for every $1 spent", a mobile menu), while an FAQ's closed details,
// a tab panel, a collapse block, a link that opens a modal
// (taylorstitch.com's "751 reviews") and a dialog's trigger
// (allrecipes.com's photo count) are.
func TestMainSkipsOverlays(t *testing.T) {
	doc := parse(t, `<html><body>
		<div id="shopify-section-sections--28184335614195__oddit_cart_drawer" class="shopify-section shopify-section-group-header-group oddit-section-active"> <div id="sections--28184335614195__oddit_cart_drawer" class="oddit-cart-drawer" style="display: none" data-cart-drawer="" aria-label="Shopping cart drawer"><p>Your cart is empty</p><a href="/products/cambridge-sunset-wheat">CAMBRIDGE Sunset Wheat</a> <span>$299.00</span></div></div>
		<div class="modal ow-modal fade" id="modal-quick-help-9" tabindex="-1" role="dialog"> <div class="modal-dialog" role="document"> <button type="button" class="close" data-dismiss="modal" aria-label="Close">x</button> <div class="modal-content"> <div class="modal-body text-left"> <h1 class="text-center">Flybuys</h1><p>Earn 1 point for every $1 spent</p></div></div></div></div>
		<div class="ow-mobile-menu"><ul><li><a href="/shop/officeworks/c/gifting">Gifts $10 and Under</a></li></ul></div>
		<dialog id="size-guide-dialog" class="dialog dialog--size-guide"><p>Size Guide 36 – XS</p></dialog>
		<main>
		<h1><span class="title">The Jack</span> <span class="subtitle">in Blue Everyday Oxford</span></h1><a class="icon-reviews review-modal-trigger" data-iframe="/products/jack-in-blue-everyday-oxford-2304?view=product-reviews" href="#"><span class="stamped-product-reviews-badge">751 reviews</span></a>
		<a id="gallery-photo_1-0" class="gallery-photo dialog-link mntl-text-link" role="button" href="#">4,926 Photos</a>
		<section class="oddit-info-page__group" id="faq-shipping-amp-delivery"> <h2 class="oddit-info-page__h2">Shipping &amp; Delivery</h2><div class="oddit-info-page__accordions"><details class="oddit-info-page__accordion"> <summary>Is shipping free?</summary> <div class="oddit-info-page__accordion-body oddit-info-page__rich"> <p>Shipping is free on orders $34.99 and up; orders under $34.99 ship for $9.</p> </div> </details></div></section>
		<div role="tabpanel" id="tab-details" hidden-until-found=""><p>100% cotton oxford cloth</p></div>
		<div class="collapse" id="faq-2"><p>Returns are accepted within 30 days.</p></div>
		</main></body></html>`)
	p := New(doc, HTMLText(doc), "https://royalcanadiancollective.com/pages/faq")
	for _, s := range []string{"$299.00", "Earn 1 point", "Gifts $10", "Size Guide"} {
		if strings.Contains(p.Main, s) {
			t.Errorf("Main holds the overlay's %q: %q", s, p.Main)
		}
	}
	for _, s := range []string{"751 reviews", "4,926 Photos", "orders under $34.99 ship for $9", "100% cotton", "within 30 days"} {
		if !strings.Contains(p.Main, s) {
			t.Errorf("Main lost the copy %q: %q", s, p.Main)
		}
	}
	if !strings.Contains(p.Text, "Earn 1 point") {
		t.Errorf("Text lost the overlay: %q", p.Text)
	}
}

// TestMainSkipsNamedChrome checks that Main leaves out what Chrome names by
// class or id: officeworks.com.au's div#header promo "Unlock a $20 OnePass
// reward*" and a OneTrust consent block, while a restaurant's div.menu in
// its main copy stays.
func TestMainSkipsNamedChrome(t *testing.T) {
	doc := parse(t, `<html><body><div id="header"><div id="ow-header"><div id="benefits-bar" class="BenefitsBarstyle__BenefitsBarMain-sc-1u86yt"><p>Unlock a $20 OnePass reward*</p></div></div></div>
		<div id="onetrust-consent-sdk"><p>We use cookies and similar technologies to collect information from your device.</p></div>
		<main><h1>Menu</h1><div class="menu"><h2>Breakfast</h2><p>Eggs on toast $18</p></div></main></body></html>`)
	p := New(doc, HTMLText(doc), "https://www.officeworks.com.au/contact-us")
	for _, s := range []string{"$20 OnePass", "We use cookies"} {
		if strings.Contains(p.Main, s) {
			t.Errorf("Main holds the chrome %q: %q", s, p.Main)
		}
	}
	if !strings.Contains(p.Main, "Eggs on toast $18") {
		t.Errorf("Main lost the menu: %q", p.Main)
	}
}
