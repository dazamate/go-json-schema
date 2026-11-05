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
	{
		// autodiscount.co.nz: one car's page, with no h1 and no cart, names
		// the car and its stock number in its address
		name: "a dealer's vehicle page",
		url:  "https://www.autodiscount.co.nz/vehicle/2015-Toyota-Aqua/12434",
		src:  autodiscountVehicle,
		want: KindProduct,
	},
	{
		// a transport agency's topic page sits under /vehicles/ too, but names
		// no vehicle (no year or stock number)
		name: "vehicles topic page",
		url:  "https://www.transport.example.gov/roads/vehicles/registration",
		src:  `<html><body><h1>Vehicle registration</h1><p>Register a vehicle for $60.</p></body></html>`,
		want: KindOther,
	},
}

// autodiscountVehicle is autodiscount.co.nz's vehicle page, trimmed: its
// price and its "You may also like" cards, which lead to other vehicles.
const autodiscountVehicle = `<html><body><div class="vehicle-price"><span class="was">was $15,950</span> $14,450 On Sale</div>
	<div class="small-12 medium-6 large-12 columns suggested"> <h6>You may also like</h6> <div class="featured-vehicles"> <ul>
	<li> <div class="small-4 columns img"> <a href="/vehicle/2015-Toyota-Aqua/12478"> <img src="/Motorcentral/VehicleData/AUT-d50de6e2-1-1.jpg" class=""> </a> </div> <div class="small-8 columns vehicle-info"> <h6 class="title">2015 Toyota Aqua 4 Grade, X-URBAN, High Spec, 4 NEW Tyres, Leather</h6> <span class="specs"> 74,750km, Automatic, Hybrid, 1500cc </span> <span class="price"> <span class="price"><span class="special"><span class="amount">$13,950</span></span></span> </span> </div> </li>
	<li> <div class="small-4 columns img"> <a href="/vehicle/2018-Nissan-Note/12471"> <img src="/Motorcentral/VehicleData/AUT-dedeffdc-1-1.jpg" class=""> </a> </div> <div class="small-8 columns vehicle-info"> <h6 class="title">2018 Nissan Note 4.5 Grade, E-Power AUTECH, 360 Camera, 4 NEW TYRES</h6> <span class="specs"> 61,920km, Automatic, Hybrid, 1200cc </span> <span class="price"> <span class="price"><span class="special"><span class="amount">$15,450</span></span></span> </span> </div> </li>
	</ul></div></div></body></html>`

// TestProductPath checks the product segments shops use: Shopify's
// /products/, Amazon's /dp/, a hardware chain's /pd/, an Asian shop's
// /goods/, a dealer's /vehicle/ with its year or stock number.
func TestProductPath(t *testing.T) {
	for path, want := range map[string]bool{
		"/products/gold-locket":               true,
		"/dp/1732046522":                      true,
		"/pd/cordless-drill/1000123":          true,
		"/goods/12345":                        true,
		"/vehicle/2015-Toyota-Aqua/12434":     true,
		"/vehicles/used/2019-ford-ranger-xlt": true,
		"/vehicles":                           false,
		"/roads/vehicles/registration":        false,
		"/vehicles/electric-vehicle-charging": false,
		"/blog/vehicle-shipping-cost":         false,
	} {
		if got := ProductPath(MustParse("https://shop.test" + path)); got != want {
			t.Errorf("ProductPath(%q) = %v, want %v", path, got, want)
		}
	}
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
		"/trolley/add":                             true, // a UK grocer's trolley
		"/addtotrolley":                            true,
		"/shop/officeworks/c/workplace-supplies/trolleys--steps-ladders/hand-trolleys": false,
	} {
		if got := CartForm.MatchString(action); got != want {
			t.Errorf("CartForm(%q) = %v, want %v", action, got, want)
		}
	}
}

// TestCartWords checks a shop that calls its cart a trolley, as UK grocers
// do ("Add to trolley", /trolley), against officeworks.com.au's and
// oliverbonas.com's trolleys for sale ("Hand Trolleys", "Drinks Trolleys").
func TestCartWords(t *testing.T) {
	for text, want := range map[string]bool{
		"Add to trolley":        true,
		"Add to my bag":         true,
		"Add it to your basket": true,
		"Add to cart":           true,
		"Hand Trolleys":         false,
		"Drinks Trolleys":       false,
		"Add to wishlist":       false,
		"Add to list":           false,
	} {
		if got := AddToCart.MatchString(text); got != want {
			t.Errorf("AddToCart(%q) = %v, want %v", text, got, want)
		}
	}
	for path, want := range map[string]bool{
		"/trolley": true,
		"/basket":  true,
		"/shop/officeworks/c/workplace-supplies/cleaning-supplies/cleaning-equipment/cleaning-trolleys": false,
	} {
		if got := CartPath.MatchString(path); got != want {
			t.Errorf("CartPath(%q) = %v, want %v", path, got, want)
		}
	}
	doc := parse(t, `<html><body><h1>Your trolley</h1><p>2 items</p></body></html>`)
	if got := New(doc, HTMLText(doc), "https://grocer.test/trolley").Kind; got != KindCart {
		t.Errorf("/trolley: Kind = %q, want %q", got, KindCart)
	}
	if got := New(doc, HTMLText(doc), "https://grocer.test/my-shop").Kind; got != KindCart {
		t.Errorf("h1 Your trolley: Kind = %q, want %q", got, KindCart)
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
