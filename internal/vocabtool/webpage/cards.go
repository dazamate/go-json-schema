package webpage

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	relatedLabel = regexp.MustCompile(`(?i)^(?:related\b|keep reading|read next|read more|more (?:for|from|on|like this|guides|articles|reading)|you (?:may|might) also like|see also|further reading|in this (?:guide|section|series|hub)\b|guides in\b)`)
	GuideLabel   = regexp.MustCompile(`(?i)^(?:in this (?:guide|section|series|hub)\b|guides in\b)`)
)

// Card is a product card on a listing: the element that holds one product's
// link and price.
type Card struct {
	N       *html.Node
	URL     string
	Name    string
	Related bool // in a block the page labels as related products
}

// SiteLink is a link's address resolved and normalized when it points at
// another page of this site.
func (p *Facts) SiteLink(a *html.Node) (string, bool) {
	h, _ := Attr(a, "href")
	h = strings.TrimSpace(h)
	if h == "" || strings.HasPrefix(h, "#") {
		return "", false
	}
	u, err := url.Parse(h)
	if err != nil || (u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	if p.Base != nil {
		u = p.Base.ResolveReference(u)
		if !strings.EqualFold(u.Hostname(), p.Base.Hostname()) {
			return "", false
		}
	} else if u.IsAbs() {
		return "", false
	}
	s := txt.NormURL(u)
	if p.URL != nil && s == txt.NormURL(p.URL) {
		return "", false
	}
	return s, true
}

// SiteLinks are the distinct pages of this site that links under n point at.
func (p *Facts) SiteLinks(n *html.Node) []string {
	var out []string
	Visit(n, func(a *html.Node) {
		if a.DataAtom == atom.A {
			if s, ok := p.SiteLink(a); ok && !slices.Contains(out, s) {
				out = append(out, s)
			}
		}
	})
	return out
}

// holds reports whether an element of the kind sits under n.
func holds(n *html.Node, a atom.Atom) bool {
	found := false
	Visit(n, func(c *html.Node) { found = found || c.DataAtom == a })
	return found
}

// ProductCards finds the page's product cards once: for each link to
// another page of the site, outside its header, footer and navs, the nearest
// element around it that shows a price and links to no other page. An
// element holding an h1 or a form is the page's own product, not a card. A
// single priced link is no listing, so it takes two.
func (p *Facts) ProductCards() []*Card {
	if p.cardsDone {
		return p.cards
	}
	p.cardsDone = true
	if p.Doc == nil {
		return nil
	}
	seen := map[*html.Node]bool{}
	var out []*Card
	Visit(p.Doc, func(a *html.Node) {
		if a.DataAtom != atom.A || Chrome(a) {
			return
		}
		u, ok := p.SiteLink(a)
		if !ok {
			return
		}
		for e, depth := a.Parent, 0; e != nil && depth < 8; e, depth = e.Parent, depth+1 {
			if e.DataAtom == atom.Body || e.DataAtom == atom.Main || holds(e, atom.H1) || holds(e, atom.Form) {
				return
			}
			if len(p.SiteLinks(e)) > 1 {
				return
			}
			if txt.PriceAmount.MatchString(HTMLText(e)) {
				if !seen[e] {
					seen[e] = true
					out = append(out, &Card{N: e, URL: u, Name: cardName(e, a)})
				}
				return
			}
		}
	})
	if len(out) < 2 {
		return nil
	}
	for _, b := range p.RelatedBoxes() {
		for _, c := range out {
			for e := c.N; e != nil; e = e.Parent {
				if e == b.N {
					c.Related = true
				}
			}
		}
	}
	p.cards = out
	return out
}

// cardName is a card's heading, else its link's text, image alt or label.
func cardName(c, a *html.Node) string {
	name := ""
	Visit(c, func(n *html.Node) {
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			if name == "" {
				name = HTMLText(n)
			}
		}
	})
	if name == "" {
		name = HTMLText(a)
	}
	if name == "" {
		for _, img := range QueryAll(a, "img[alt]") {
			name, _ = Attr(img, "alt")
		}
	}
	if name == "" {
		name, _ = Attr(a, "aria-label")
	}
	// a link that wraps the whole card carries its price too
	return strings.Join(strings.Fields(txt.PriceAmount.ReplaceAllString(name, "")), " ")
}

// InCard reports whether n sits in a product card. It finds the page's
// cards first, so the answer never depends on what ran before it.
func (p *Facts) InCard(n *html.Node) bool {
	p.ProductCards()
	return p.inFoundCard(n)
}

// inFoundCard reports whether n sits in a product card already found. It
// finds none itself: classify runs it before the cards can be known.
func (p *Facts) inFoundCard(n *html.Node) bool {
	if !p.cardsDone {
		return false
	}
	for _, c := range p.cards {
		for e := n; e != nil; e = e.Parent {
			if e == c.N {
				return true
			}
		}
	}
	return false
}

// RelatedBox is a block the page labels as related reading or products.
type RelatedBox struct {
	Label string
	N     *html.Node
}

// RelatedBoxes finds the blocks a short label ("You may also like", "Keep
// reading") heads: the nearest section, aside, nav or article around the
// label, else the nearest element around it that links to other pages.
func (p *Facts) RelatedBoxes() []RelatedBox {
	if p.Doc == nil {
		return nil
	}
	var out []RelatedBox
	Visit(p.Doc, func(n *html.Node) {
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.P, atom.Span:
		default:
			return
		}
		t := strings.TrimSpace(HTMLText(n))
		// a guide's own nav of its pages sits in the content, not the site's
		// chrome
		if len(strings.Fields(t)) > 6 || !relatedLabel.MatchString(t) || (Chrome(n) && !(GuideLabel.MatchString(t) && !siteChromeAround(n))) {
			return
		}
		for e := n.Parent; e != nil && e.DataAtom != atom.Body && e.DataAtom != atom.Main; e = e.Parent {
			switch e.DataAtom {
			case atom.Section, atom.Aside, atom.Nav, atom.Article:
			default:
				if len(p.SiteLinks(e)) == 0 {
					continue
				}
			}
			if !slices.ContainsFunc(out, func(b RelatedBox) bool { return b.N == e }) {
				out = append(out, RelatedBox{t, e})
			}
			return
		}
	})
	return out
}

// Sells reports a page of a shop: an add-to-cart form, a link to its cart or
// checkout, a cart button, a shop platform's page, or product cards that
// lead to product pages. Cards alone are no shop: a SaaS plan, a service or
// a home for sale has a price and a link too.
func (p *Facts) Sells() bool {
	if p.Doc == nil {
		return false
	}
	if p.BuyForm() != nil || shopPlatform(p.Doc) {
		return true
	}
	for _, c := range p.ProductCards() {
		if ProductPath(MustParse(c.URL)) {
			return true
		}
	}
	for _, a := range QueryAll(p.Doc, "a[href]") {
		if u, ok := p.SiteLink(a); ok && (CartPath.MatchString(u) || checkoutPath.MatchString(u)) {
			return true
		}
	}
	for _, b := range QueryAll(p.Doc, "a, button") {
		if CartControl(b) {
			return true
		}
	}
	return false
}

var (
	checkoutPath = regexp.MustCompile(`/checkouts?$`)
	// cartWord is a word that names the cart; controlFiller the words a cart
	// button's name wraps it in ("Open cart", "Basket quantity 0", "Shopping
	// Cart Menu 0 items in the cart").
	cartWord      = regexp.MustCompile(`^(?:cart|basket|bag|trolley|minicart|minibasket|minibag)$`)
	controlFiller = regexp.MustCompile(`^(?:\d+|open|view|show|go|goto|to|your|my|the|shopping|mini|menu|icon|link|quantity|click|items?|with|in|drawer|button|and|of|total)$`)
)

// CartControl reports a link or button for the cart: its name (aria-label,
// else its text, else its title) is the cart's, in a cart button's words. A
// link named "Bag Straps" sells bags; one named "Bag" opens the cart.
func CartControl(n *html.Node) bool {
	name, _ := Attr(n, "aria-label")
	if strings.TrimSpace(name) == "" {
		name = HTMLText(n)
	}
	if strings.TrimSpace(name) == "" {
		name, _ = Attr(n, "title")
	}
	words := strings.Fields(txt.NormWords(name))
	cart := false
	for _, w := range words {
		switch {
		case cartWord.MatchString(w):
			cart = true
		case !controlFiller.MatchString(w):
			return false
		}
	}
	return cart
}

// shopPlatform reports a page of a shop platform's store: Shopify's digital
// wallet meta or its Shopify.shop script, or WooCommerce's body class.
func shopPlatform(doc *html.Node) bool {
	found := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil && !found; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			switch c.DataAtom {
			case atom.Meta:
				name, _ := Attr(c, "name")
				found = name == "shopify-digital-wallet" || name == "shopify-checkout-api-token"
			case atom.Script:
				found = c.FirstChild != nil && strings.Contains(c.FirstChild.Data, "Shopify.shop")
			case atom.Body:
				class, _ := Attr(c, "class")
				found = slices.ContainsFunc(strings.Fields(class), func(s string) bool { return strings.HasPrefix(s, "woocommerce") })
			}
			walk(c)
		}
	}
	walk(doc)
	return found
}

// Promo reports a shop's dl that sells rather than defines: one whose block
// links into a category or collection ("Shop lockets").
func (p *Facts) Promo(dl *html.Node) bool {
	if !p.Sells() {
		return false
	}
	for e, up := dl.Parent, 0; e != nil && up < 2 && e.DataAtom != atom.Main && e.DataAtom != atom.Body; e, up = e.Parent, up+1 {
		for _, a := range QueryAll(e, "a[href]") {
			if u, ok := p.SiteLink(a); ok && listingLink(MustParse(u)) {
				return true
			}
		}
	}
	return false
}

// listingLink reports an address of a category or collection: its first
// segment after any locale names one (/au/collections/lockets).
func listingLink(u *url.URL) bool {
	segs := Segments(u)
	if len(segs) > 0 && LocaleSegment.MatchString(segs[0]) {
		segs = segs[1:]
	}
	return len(segs) > 1 && listingSegment.MatchString(segs[0])
}

// HomeLink reports a link to the site's home page: its root, or a locale root
// such as /au.
func (p *Facts) HomeLink(a *html.Node) bool {
	h, _ := Attr(a, "href")
	u, err := url.Parse(strings.TrimSpace(h))
	if err != nil || h == "" {
		return false
	}
	if p.Base != nil {
		u = p.Base.ResolveReference(u)
		if !strings.EqualFold(u.Hostname(), p.Base.Hostname()) {
			return false
		}
	}
	segs := Segments(u)
	return len(segs) == 0 || (len(segs) == 1 && LocaleSegment.MatchString(segs[0]))
}
