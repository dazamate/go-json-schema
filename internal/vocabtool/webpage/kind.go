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
	{KindContact, regexp.MustCompile(`^contact(-us)?$`)},
	{KindAbout, regexp.MustCompile(`^(about|about-us|our-story)$`)},
	{KindShipping, regexp.MustCompile(`^(shipping|delivery|postage)(-(policy|information|info))?$`)},
	{KindReturns, regexp.MustCompile(`^(returns?|refunds?|exchanges?)(-(policy|information|info))?$|^(return|refund)-policy$`)},
	{KindFAQ, regexp.MustCompile(`^(faqs?|frequently-asked-questions)$`)},
}

var (
	productSegment = regexp.MustCompile(`^(products?|p|item|items)$`)
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
	cartPath  = regexp.MustCompile(`/(cart|basket|bag)$`)
	CartForm  = regexp.MustCompile(`(?i)(cart|basket|bag)\b`)
	AddToCart = regexp.MustCompile(`(?i)\b(add to (cart|bag|basket)|buy now|pre-?order)\b`)
)

// Segments are a URL path's segments, lowercased.
func Segments(u *url.URL) []string {
	if u == nil {
		return nil
	}
	return strings.FieldsFunc(strings.ToLower(u.Path), func(r rune) bool { return r == '/' })
}

// BuyForm is the page's add-to-cart form outside product cards, or nil.
func (p *Facts) BuyForm() *html.Node {
	if p.Doc == nil {
		return nil
	}
	var form *html.Node
	Visit(p.Doc, func(n *html.Node) {
		if form != nil || n.DataAtom != atom.Form || p.InCard(n) {
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
	for _, ks := range kindSegments {
		if slices.ContainsFunc(segs, ks.re.MatchString) {
			return ks.kind
		}
	}
	if p.Doc != nil && strings.HasPrefix(strings.ToLower(MetaContent(p.Doc, "property", "og:type")), "product") {
		return KindProduct
	}
	if i := slices.IndexFunc(segs, productSegment.MatchString); i >= 0 && i < len(segs)-1 {
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
	if (slices.ContainsFunc(segs, listingSegment.MatchString) && listed >= 2) || listed >= 3 {
		return KindListing
	}
	return KindOther
}

// privateKinds are pages search engines don't index, so the shop checks
// leave them be.
var privateKinds = map[Kind]string{KindCart: "a shopping cart", KindCheckout: "a checkout", KindAccount: "an account page"}
