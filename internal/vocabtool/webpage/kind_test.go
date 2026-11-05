package webpage

import "testing"

// kindRows are real snippets of corpus pages, trimmed to what decides the
// page's kind.
var kindRows = []struct {
	name, url, src string
	want           Kind
}{
	{
		// cultfurniture.com: a Shopify collection says og:type product.group,
		// a group of products, and its cart drawer's form posts to /cart
		name: "collection with a cart drawer",
		url:  "https://www.cultfurniture.com/collections/sale",
		src: `<html><head><meta property="og:type" content="product.group"></head><body>
			<header role="banner"><a href="/cart">0 Cart</a></header>
			<h1 class="clc-Hero_Title" id="sale">Sale</h1>
			<site-drawer key="cart"><form class="psh-CartFooter_Buttons" action="/cart" method="POST">
			<button type="submit" name="checkout" id="checkout"> Checkout Securely </button>
			<a href="/cart">View cart</a></form></site-drawer></body></html>`,
		want: KindListing,
	},
	{
		// kasaustralia.com.au: the product form posts to /cart/add
		name: "product form posts to cart/add",
		url:  "https://www.kasaustralia.com.au/products/belrose-cushion-charcoal",
		src: `<html><body><h1>Belrose Cushion</h1><p>$67.95 AUD</p>
			<form method="post" action="/cart/add" id="AddToCartForm--template__product" class="product__form">
			<select name="id"><option>Charcoal</option><option>Mustard</option></select>
			<button type="submit" name="add">Add to Cart</button></form></body></html>`,
		want: KindProduct,
	},
	{
		// a page of a shop with the cultfurniture cart drawer is not a product
		// page for having it
		name: "cart drawer on a content page",
		url:  "https://www.cultfurniture.com/pages/students",
		src: `<html><body><h1>Students</h1><p>Student discount.</p>
			<form class="psh-CartFooter_Buttons" action="/cart" method="POST">
			<button type="submit" name="checkout" id="checkout"> Checkout Securely </button>
			<a href="/cart">View cart</a></form></body></html>`,
		want: KindOther,
	},
}

// TestClassifyAddress checks the kinds a page's address gives, with no
// heading to go by: verizon.com's contactus, oliverbonas.com's
// returns-and-refunds, and the underscore and file-name slugs older sites
// use.
func TestClassifyAddress(t *testing.T) {
	for u, want := range map[string]Kind{
		"https://www.verizon.com/digital/nsa/nos/ui/ngd/contactus":            KindContact,
		"https://www.oliverbonas.com/returns-and-refunds":                     KindReturns,
		"https://www.taylorstitch.com/policies/shipping-policy":               KindShipping,
		"https://royalcanadiancollective.com/pages/faq":                       KindFAQ,
		"https://shop.test/pages/shipping-and-returns":                        KindShipping,
		"https://shop.test/contact_us.php":                                    KindContact,
		"https://shop.test/refund_policy.html":                                KindReturns,
		"https://www.tutorfinder.com.au/help/tutor_rates.php":                 KindOther,
		"https://business.gov.au/Registrations/Register-licences-and-permits": KindOther,
	} {
		doc := parse(t, `<html><body><main><p>Text.</p></main></body></html>`)
		if got := New(doc, HTMLText(doc), u).Kind; got != want {
			t.Errorf("%s: Kind = %q, want %q", u, got, want)
		}
	}
}

// TestClassifyCards checks that priced cards make a listing only on a site
// that sells: a SaaS's plans, a pest controller's services and the homes in a
// real estate blog post are no shop's listing.
func TestClassifyCards(t *testing.T) {
	want := map[string]Kind{
		"SaaS plan cards":              KindOther,
		"service cards":                KindOther,
		"home listings in a blog post": KindOther,
	}
	for _, r := range sellsRows {
		w, ok := want[r.name]
		if !ok {
			continue
		}
		doc := parse(t, r.src)
		if got := New(doc, HTMLText(doc), r.url).Kind; got != w {
			t.Errorf("%s: Kind = %q, want %q", r.name, got, w)
		}
	}
	// royalcanadiancollective.com cards, on a collection of the shop
	doc := parse(t, `<html><body><h1>Women's boots</h1>
		<div class="card"><a href="/products/cambridge-sunset-wheat-womens-waterproof-insulated">CAMBRIDGE Sunset Wheat</a><span>$289.00</span></div>
		<div class="card"><a href="/products/dalhousie-womens-winter-boot-sunset-wheat">DALHOUSIE Sunset Wheat</a><span>$299.00</span></div>
		<div class="card"><a href="/products/cambridge-black-lager-womens">CAMBRIDGE Black Lager</a><span>$289.00</span></div>
		</body></html>`)
	if got := New(doc, HTMLText(doc), "https://royalcanadiancollective.com/collections/womens").Kind; got != KindListing {
		t.Errorf("product cards: Kind = %q, want %q", got, KindListing)
	}
}

func TestCartForm(t *testing.T) {
	for action, want := range map[string]bool{
		"/cart/add":                                true, // Shopify
		"/product/candle/?add-to-cart=17302":       true, // WooCommerce
		"/checkout/cart/add/uenc/aHR0/product/12/": true, // Magento
		"/basket/add":                              true,
		"/cart":                                    false, // the cart drawer's checkout
		"/cart/checkout":                           false,
		"/collections/beanbag-chairs":              false,
	} {
		if got := CartForm.MatchString(action); got != want {
			t.Errorf("CartForm(%q) = %v, want %v", action, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	for _, r := range kindRows {
		doc := parse(t, r.src)
		if got := New(doc, HTMLText(doc), r.url).Kind; got != r.want {
			t.Errorf("%s: Kind = %q, want %q", r.name, got, r.want)
		}
	}
}
