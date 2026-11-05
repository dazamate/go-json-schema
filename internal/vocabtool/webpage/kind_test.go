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
