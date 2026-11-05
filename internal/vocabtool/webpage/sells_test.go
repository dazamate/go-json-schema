package webpage

import "testing"

// sellsRows are real snippets of corpus pages, trimmed to what says whether
// the site sells online.
var sellsRows = []struct {
	name, url, src string
	want           bool
}{
	{
		// oliverbonas.com: the basket is a button, not a link
		name: "basket button",
		url:  "https://www.oliverbonas.com/returns-and-refunds",
		src: `<html><body><header><button baskettoggle="" class="_icon basket-qty pos-relative" aria-label="Basket quantity 0 click to open minibasket"><i aria-hidden="true" class="icon"></i></button></header>
			<h1>Returns and refunds</h1></body></html>`,
		want: true,
	},
	{
		// officeworks.com.au: the cart button's text says Cart
		name: "cart button text",
		url:  "https://www.officeworks.com.au/contact-us",
		src: `<html><body><ul><li id="header-cart-tile" class="style__SuperCartTileWrap"><button aria-expanded="false" class="style__SuperCartTileButton"><svg name="base--icons--cart" focusable="false" aria-hidden="true"></svg><span class="style__CartLabel">0</span><div class="style__SuperCartTileText">Cart</div></button></li></ul>
			<h1>Contact us</h1></body></html>`,
		want: true,
	},
	{
		// verizon.com: a shopping cart menu button
		name: "shopping cart menu",
		url:  "https://www.verizon.com/support/contact-us/",
		src: `<html><body><div class="gnav20-utility-wrapper" item-title="unifiedCart"><button id="gnav20-cart-icon" class="gnav20-cart" aria-label="Shopping Cart Menu 0 items in the cart" aria-expanded="false"></button></div>
			<h1>Contact us</h1></body></html>`,
		want: true,
	},
	{
		// royalcanadiancollective.com: a link to the checkout
		name: "checkout link",
		url:  "https://royalcanadiancollective.com/pages/faq",
		src:  `<html><body><h1>FAQ</h1><a href="/cart/checkout" class="button">Secure Checkout</a></body></html>`,
		want: true,
	},
	{
		// a Shopify store's pages carry its digital wallet meta
		name: "Shopify store",
		url:  "https://royalcanadiancollective.com/pages/faq",
		src:  `<html><head><meta id="shopify-digital-wallet" name="shopify-digital-wallet" content="/58763247859/digital_wallets/dialog"></head><body><h1>FAQ</h1></body></html>`,
		want: true,
	},
	{
		// sublimecandles.co.uk: WooCommerce marks the body
		name: "WooCommerce store",
		url:  "https://sublimecandles.co.uk/about/",
		src:  `<html><body class="wp-singular page-template-default page et-tb-has-header woocommerce-js et_divi_theme"><h1>About</h1></body></html>`,
		want: true,
	},
	{
		// thegymgroup.com: a gym's join basket, no cart
		name: "return user basket",
		url:  "https://www.thegymgroup.com/gym-membership/",
		src:  `<html><body><a href="/login/">Account</a><button aria-label="Return User Basket" type="button" class="css-17spkya"></button><h1>Gym membership</h1></body></html>`,
		want: false,
	},
	{
		// oliverbonas.com: bags are things it sells, not its cart
		name: "bag category links",
		url:  "https://www.oliverbonas.com/returns-and-refunds",
		src:  `<html><body><h1>Returns</h1><a href="/accessories/bags/bag-straps">Bag Straps</a><a href="/accessories/keyrings">Keyrings &amp; Bag Charms</a></body></html>`,
		want: false,
	},
	{
		// notion.com/pricing: plan cards lead to the signup, not to products
		name: "SaaS plan cards",
		url:  "https://www.notion.com/pricing",
		src: `<html><body><h1>One tool for your whole company.</h1>
			<div class="AkOUza_header"><h3>Free</h3><div class="AkOUza_pricingWrap"><span>$0</span><span>per seat / month</span></div><p>For individuals to organize personal projects and life.</p><div class="AkOUza_ctaWrap"><a href="/signup">Sign up</a></div></div>
			<div class="AkOUza_header"><h3>Plus</h3><div class="AkOUza_pricingWrap"><span>$10</span><span>per seat / month</span></div><p>For small teams.</p><div class="AkOUza_ctaWrap"><a href="/signup?target=plans">Get started</a></div></div>
			<div class="AkOUza_header"><h3>Business</h3><div class="AkOUza_pricingWrap"><span>$20</span><span>per seat / month</span></div><p>For growing businesses.</p><div class="AkOUza_ctaWrap"><a href="/signup?target=business">Get started</a></div></div>
			</body></html>`,
		want: false,
	},
	{
		// pestproof.co.nz/pricing: service cards
		name: "service cards",
		url:  "https://pestproof.co.nz/pricing/",
		src: `<html><body><h1>Pricing</h1>
			<div class="fusion-builder-row"><h2><a href="https://pestproof.co.nz/ant-treatment/">Ant Treatment</a></h2><p><strong>Our ant treatment service is available for $300 + gst and includes:</strong></p></div>
			<div class="fusion-builder-row"><h2><a href="https://pestproof.co.nz/bed-bug-treatment/">Bed Bug Treatment</a></h2><p><strong>Our bed bug treatment is available for $450 + gst</strong></p></div>
			<div class="fusion-builder-row"><h2><a href="https://pestproof.co.nz/flea-treatment/">Flea Treatment</a></h2><p><strong>Our flea treatment is available for $250 + gst</strong></p></div>
			</body></html>`,
		want: false,
	},
	{
		// redfin.com/blog: homes for sale in an article
		name: "home listings in a blog post",
		url:  "https://www.redfin.com/blog/checklist-for-buying-a-house/",
		src: `<html><body><article><h1>First-Time Home Buyer Checklist</h1>
			<div class="house-listing"><div class="photo"><a href="https://www.redfin.com/IL/Chicago/4130-N-Kenneth-Ave-60641/home/22650100?utm_source=blog"><img src="a.jpg" alt="Home Image"></a></div><div class="price"><strong>$1,850,000</strong></div><div class="details">5 beds, 3.5 baths, 4228 sq ft</div></div>
			<div class="house-listing"><div class="photo"><a href="https://www.redfin.com/IL/Schaumburg/1425-Concord-Ln-60193/home/13716465?utm_source=blog"><img src="b.jpg" alt="Home Image"></a></div><div class="price"><strong>$1,799,000</strong></div><div class="details">4 beds, 3 baths, 1936 sq ft</div></div>
			</article></body></html>`,
		want: false,
	},
	{
		// royalcanadiancollective.com: product cards lead to product pages
		name: "product cards",
		url:  "https://royalcanadiancollective.com/pages/faq",
		src: `<html><body><h1>FAQ</h1>
			<div class="card"><a href="/products/cambridge-sunset-wheat-womens-waterproof-insulated">CAMBRIDGE Sunset Wheat</a><span>$289.00</span></div>
			<div class="card"><a href="/products/dalhousie-womens-winter-boot-sunset-wheat">DALHOUSIE Sunset Wheat</a><span>$299.00</span></div>
			</body></html>`,
		want: true,
	},
}

func TestSells(t *testing.T) {
	for _, r := range sellsRows {
		doc := parse(t, r.src)
		if got := New(doc, HTMLText(doc), r.url).Sells(); got != r.want {
			t.Errorf("%s: Sells() = %v, want %v", r.name, got, r.want)
		}
	}
}
