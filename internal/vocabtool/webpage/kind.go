package webpage

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Kind is what a shop's page is for.
type Kind string

const (
	KindOther    Kind = ""
	KindHome     Kind = "home"
	KindProduct  Kind = "product"
	KindListing  Kind = "listing"
	KindSearch   Kind = "search"
	KindContact  Kind = "contact"
	KindAbout    Kind = "about"
	KindShipping Kind = "shipping"
	KindReturns  Kind = "returns"
	KindFAQ      Kind = "faq"
	KindCart     Kind = "cart"
	KindCheckout Kind = "checkout"
	KindAccount  Kind = "account"
)

// kindSegments are the URL path segments that say what a page is.
var kindSegments = []struct {
	kind Kind
	re   *regexp.Regexp
}{
	{KindCart, regexp.MustCompile(`^(cart|basket|bag|shopping-bag|shopping-cart)$`)},
	{KindCheckout, regexp.MustCompile(`^checkouts?$`)},
	{KindAccount, regexp.MustCompile(`^(account|my-account|login|signin|sign-in|register|signup|sign-up|wishlist|wish-list|favourites|favorites)$`)},
	{KindSearch, regexp.MustCompile(`^search$`)},
	{KindContact, regexp.MustCompile(`^contact(-?us)?$`)},
	{KindAbout, regexp.MustCompile(`^(about|about-us|our-story)$`)},
	{KindShipping, regexp.MustCompile(`^(shipping|delivery|postage)(-(and|&)-(returns?|refunds?))?(-(policy|information|info))?$`)},
	{KindReturns, regexp.MustCompile(`^(returns?|refunds?|exchanges?)(-(and|&)-(returns?|refunds?|exchanges?))?(-(policy|information|info))?$`)},
	{KindFAQ, regexp.MustCompile(`^(faqs?|frequently-asked-questions)$`)},
}

var (
	productSegment = regexp.MustCompile(`^(products?|p|item|items)$`)
	ogProduct      = regexp.MustCompile(`(?i)^\s*product(?:\.item)?\s*$`)
	fileExt        = regexp.MustCompile(`\.(?:html?|php\d?|aspx?|jsp|cfm)$`)
	listingSegment = regexp.MustCompile(`^(categor(y|ies)|collections?|c|shop|store|catalog(ue)?|departments?)$`)
	LocaleSegment  = regexp.MustCompile(`^[a-z]{2}(-[a-z]{2})?$`)
	kindHeadings   = []struct {
		kind Kind
		re   *regexp.Regexp
	}{
		{KindSearch, regexp.MustCompile(`(?i)^search results|^results for`)},
		{KindCart, regexp.MustCompile(`(?i)^your (bag|cart|basket)\b`)},
		{KindCheckout, regexp.MustCompile(`(?i)^checkout\b`)},
		{KindContact, regexp.MustCompile(`(?i)^contact\b`)},
		{KindAbout, regexp.MustCompile(`(?i)^about\b|^our story\b`)},
		{KindShipping, regexp.MustCompile(`(?i)^(shipping|delivery|postage)\b`)},
		{KindReturns, regexp.MustCompile(`(?i)^(returns?|refunds?)\b`)},
		{KindFAQ, regexp.MustCompile(`(?i)^(faqs?|frequently asked)\b`)},
	}
	// CartPath matches the address of the cart page (/cart, /basket, /bag).
	CartPath = regexp.MustCompile(`/(cart|basket|bag)$`)
	// CartForm matches the action of a form that adds to the cart
	// (/cart/add, ?add-to-cart=, addtobasket): a form that only posts to the
	// cart is the cart's own (its checkout button), not a product's.
	CartForm  = regexp.MustCompile(`(?i)\b(?:cart|basket|bag)[/_-]?add\b|\badd[/_-]?to[/_-]?(?:cart|basket|bag)\b|\badd(?:to)?(?:cart|basket|bag)\b`)
	AddToCart = regexp.MustCompile(`(?i)\b(add to (cart|bag|basket)|buy now|pre-?order)\b`)

	// PreOrder finds a shop taking pre-orders ("Pre-order now").
	PreOrder = regexp.MustCompile(`(?i)\bpre-?order(?: now)?\b`)
)

// Segments are a URL path's segments, lowercased.
func Segments(u *url.URL) []string {
	if u == nil {
		return nil
	}
	return strings.FieldsFunc(strings.ToLower(u.Path), func(r rune) bool { return r == '/' })
}

// ProductPath reports the address of a product page: a product segment with
// the product's own after it (/products/gold-locket, /p/12345).
func ProductPath(u *url.URL) bool {
	segs := Segments(u)
	i := slices.IndexFunc(segs, productSegment.MatchString)
	return i >= 0 && i < len(segs)-1
}

// categorySegment names a category or collection in an address.
var categorySegment = regexp.MustCompile(`^(categor(y|ies)|collections?|c|departments?)$`)

// categoryLink reports the address of a category or collection page
// (/collections/gifts-under-30, /shop/officeworks/c/gifting), not a product's
// (/collections/sale/products/stool).
func categoryLink(u *url.URL) bool {
	return !ProductPath(u) && slices.ContainsFunc(Segments(u), categorySegment.MatchString)
}

// BuyForm is the page's add-to-cart form outside product cards, or nil.
func (p *Facts) BuyForm() *html.Node {
	if p.Doc == nil {
		return nil
	}
	var form *html.Node
	Visit(p.Doc, func(n *html.Node) {
		if form != nil || n.DataAtom != atom.Form || p.inFoundCard(n) {
			return
		}
		action, _ := Attr(n, "action")
		if CartForm.MatchString(action) || AddToCart.MatchString(HTMLText(n)) {
			form = n
		}
	})
	return form
}

// classify works out what kind of page this is: from its address, then its
// og:type, add-to-cart form and product cards, then its h1.
func (p *Facts) classify() Kind {
	segs := Segments(p.URL)
	// a slug as older sites write it: contact_us.php, refund_policy.html
	slugs := make([]string, len(segs))
	for i, s := range segs {
		slugs[i] = strings.ReplaceAll(fileExt.ReplaceAllString(s, ""), "_", "-")
	}
	for _, ks := range kindSegments {
		if slices.ContainsFunc(slugs, ks.re.MatchString) {
			return ks.kind
		}
	}
	// og:type product.group is a group of products: a collection, or a
	// product's variants, which its add-to-cart form tells apart below
	if p.Doc != nil && ogProduct.MatchString(MetaContent(p.Doc, "property", "og:type")) {
		return KindProduct
	}
	if ProductPath(p.URL) {
		return KindProduct
	}
	h := p.H1()
	for _, kh := range kindHeadings {
		if kh.re.MatchString(h) {
			return kh.kind
		}
	}
	if p.BuyForm() != nil && h != "" {
		return KindProduct
	}
	if len(segs) == 0 || (len(segs) == 1 && LocaleSegment.MatchString(segs[0])) {
		if p.URL != nil {
			return KindHome
		}
	}
	listed := 0
	for _, c := range p.ProductCards() {
		if !c.Related {
			listed++
		}
	}
	// a shop's listing: its cards, or a category or collection address whose
	// grid the page loads after it renders. Priced cards on a site that sells
	// nothing are plans, services or homes, not a shop's listing.
	if ((slices.ContainsFunc(segs, listingSegment.MatchString) && listed >= 2) || listed >= 3 || listingLink(p.URL)) && p.Sells() {
		return KindListing
	}
	return KindOther
}

// privateKinds are pages search engines don't index, so the shop checks
// leave them be.
var privateKinds = map[Kind]string{KindCart: "a shopping cart", KindCheckout: "a checkout", KindAccount: "an account page"}
