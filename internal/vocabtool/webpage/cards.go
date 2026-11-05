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
	relatedLabel = regexp.MustCompile(`(?i)^(?:related\b|keep reading|read next|read more|more (?:for|from|on|like this|guides|articles|reading)|you (?:may|might) also like|see also|further reading|in this (?:guide|section|series|hub)\b|guides in\b|recommendations\b|recommended (?:for|products?|items?|reading|articles?|posts?|guides?)\b|recently viewed|(?:this )?goes well with|(?:customers|people) (?:also|who)\b|frequently bought|similar\b|complete the look|shop the look|pairs? (?:well )?with|(?:wear|style) it with)`)
	GuideLabel   = regexp.MustCompile(`(?i)^(?:(?:(?:pages|parts|chapters|articles|steps) )?in this (?:guide|section|series|hub)\b|guides in\b)`)
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
// another page of the site (not a photo or a document), outside its header,
// footer and navs, the nearest element around it that shows a price and
// links to no other page (a link with another query is the same page: a
// recommender's tracking). An element
// that is or holds a form or holds an h1 is the page's own product, not a
// card. A card shows a product's price: a sentence or table cell with a
// price and a link is copy, and a price bracket ("Gifts $20 and Under") or a
// plan's rate ("$10 per seat / month") is no product's price. A single
// priced link is no listing, so it takes two.
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
		if a.DataAtom != atom.A || Chrome(a) || Overlay(a) || StyleHidden(a) {
			return
		}
		u, ok := p.SiteLink(a)
		if !ok || categoryLink(MustParse(u)) || mediaFile.MatchString(pageOf(u)) || slices.ContainsFunc(out, func(c *Card) bool { return pageOf(c.URL) == pageOf(u) }) {
			return
		}
		for e, depth := a.Parent, 0; e != nil && depth < 8; e, depth = e.Parent, depth+1 {
			if e.DataAtom == atom.Body || e.DataAtom == atom.Main || e.DataAtom == atom.Form || holds(e, atom.H1) || holds(e, atom.Form) {
				return
			}
			if p.pagesUnder(e) > 1 {
				return
			}
			if t := HTMLText(e); txt.PriceAmount.MatchString(t) {
				if copyBlock[e.DataAtom] || !productPrice(t) {
					return
				}
				if !seen[e] {
					seen[e] = true
					out = append(out, &Card{N: e, URL: u, Name: p.cardName(e, a, u)})
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

// copyBlock are the elements of copy, not cards: a paragraph, a table cell
// or a definition that gives a price beside a link ("Free Shipping over $150
// spend. View more details HERE").
var copyBlock = map[atom.Atom]bool{atom.P: true, atom.Td: true, atom.Th: true, atom.Dd: true, atom.Label: true, atom.Blockquote: true, atom.Figcaption: true}

var (
	// bracketBefore and bracketAfter are the words that make a price a
	// bracket of prices: "GIFTS UNDER $50", "Gifts $20 and Under".
	bracketBefore = regexp.MustCompile(`(?i)\b(?:under|below|less than|up to)\s*$`)
	bracketAfter  = regexp.MustCompile(`(?i)^\s*(?:and|or|&)\s+(?:under|below|less)\b`)
	// planRate is a plan's period after its price: "$10 per seat / month",
	// "$8/mo", "$96 a year", "billed annually".
	planRate = regexp.MustCompile(`(?i)^\s*(?:(?:/|per|a|an)\s*(?:(?:seat|user|member|person|licen[cs]e)\s*(?:/|per|a)?\s*)?(?:mo|mth|month|yr|year|annum|wk|week)\b|(?:monthly|annually|yearly|billed)\b)`)
)

// productPrice reports a price in t that is a product's: not a bracket of
// prices and not a plan's rate.
func productPrice(t string) bool {
	for _, loc := range txt.PriceAmount.FindAllStringIndex(t, -1) {
		before, after := t[:loc[0]], t[loc[1]:]
		if !bracketBefore.MatchString(before) && !bracketAfter.MatchString(after) && !planRate.MatchString(after) {
			return true
		}
	}
	return false
}

// PlanPrices counts the plan prices in t: prices a plan's rate follows ("$10
// per seat / month", "$8/mo", "$96 a year").
func PlanPrices(t string) int {
	n := 0
	for _, loc := range txt.PriceAmount.FindAllStringIndex(t, -1) {
		if planRate.MatchString(t[loc[1]:]) {
			n++
		}
	}
	return n
}

// mediaFile is the address of a photo, video or document, not a page: a
// gallery's link to a vehicle's photo is no product.
var mediaFile = regexp.MustCompile(`(?i)\.(?:jpe?g|png|gif|webp|avif|svg|pdf|mp4|webm)$`)

// pageOf is a site link's page: its address without its query and fragment.
func pageOf(u string) string {
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		return u[:i]
	}
	return u
}

// pagesUnder counts the distinct pages of this site that links under n
// point at.
func (p *Facts) pagesUnder(n *html.Node) int {
	var pages []string
	for _, s := range p.SiteLinks(n) {
		if s = pageOf(s); !slices.Contains(pages, s) {
			pages = append(pages, s)
		}
	}
	return len(pages)
}

// callToAction is a card link's words that name what it does, not the
// product: "Quick Buy", "Shop now", "Add to cart".
var callToAction = regexp.MustCompile(`(?i)^(?:quick (?:buy|view|shop|add)|shop now|buy now|view(?: details| product)?|add to (?:cart|bag|basket|trolley)|select options|choose options|learn more|more info|details|see more)$`)

// cardName is a card's heading, else the text of its links to the product
// that is no call to action: its title link's before an image link's (a
// card often links its image, described for a screen reader, and its
// title), else its link's image alt or label.
func (p *Facts) cardName(c, a *html.Node, u string) string {
	name := ""
	Visit(c, func(n *html.Node) {
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			if name == "" {
				name = HTMLText(n)
			}
		}
	})
	for _, images := range []bool{false, true} {
		Visit(c, func(n *html.Node) {
			if s, ok := p.SiteLink(n); ok && name == "" && n.DataAtom == atom.A && pageOf(s) == pageOf(u) && holds(n, atom.Img) == images && !callToAction.MatchString(HTMLText(n)) {
				name = HTMLText(n)
			}
		})
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
	var named []*html.Node  // the boxes their aria-label names
	var titled []*html.Node // the blocks around the page's h1
	Visit(p.Doc, func(n *html.Node) {
		if titled == nil && n.DataAtom == atom.H1 {
			titled = Ancestors(n)
		}
	})
	Visit(p.Doc, func(n *html.Node) {
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.P, atom.Span:
		case atom.Nav, atom.Aside, atom.Section:
			// a guide's nav its label names, over a "Contents" heading
			// (gov.uk's "Pages in this guide")
			if l, _ := Attr(n, "aria-label"); GuideLabel.MatchString(strings.TrimSpace(l)) && !siteChromeAround(n) && len(p.SiteLinks(n)) >= 2 && !slices.ContainsFunc(out, func(b RelatedBox) bool { return b.N == n }) {
				out = append(out, RelatedBox{strings.TrimSpace(l), n})
				named = append(named, n)
			}
			return
		default:
			return
		}
		t := strings.TrimSpace(HTMLText(n))
		// a guide's own nav of its pages sits in the content, not the site's
		// chrome
		// a button's or link's text ("Read more" on a description's toggle)
		// labels the control, not a box of links
		if len(strings.Fields(t)) > 6 || !relatedLabel.MatchString(t) || (Chrome(n) && !(GuideLabel.MatchString(t) && !siteChromeAround(n))) || inControl(n) || rowKey(n) {
			return
		}
		// the nearest block with a link of the site's is the box when the
		// climb reaches the page's own copy
		var first *html.Node
		for e := n.Parent; e != nil && e.DataAtom != atom.Body && e.DataAtom != atom.Main; e = e.Parent {
			// a related box never holds the page's own title
			if slices.Contains(titled, e) {
				if e = first; e == nil {
					return
				}
			} else {
				switch e.DataAtom {
				case atom.Section, atom.Aside, atom.Nav, atom.Article:
				default:
					k := len(p.SiteLinks(e))
					if k > 0 && first == nil {
						first = e
					}
					// a box of links, not a header row with its one "View all"
					if k < 2 {
						continue
					}
				}
			}
			if !slices.ContainsFunc(out, func(b RelatedBox) bool { return b.N == e }) {
				out = append(out, RelatedBox{t, e})
			}
			return
		}
	})
	// a box its label names gives way to the box its own heading gives
	return slices.DeleteFunc(out, func(b RelatedBox) bool {
		return slices.Contains(named, b.N) && slices.ContainsFunc(out, func(o RelatedBox) bool { return o.N != b.N && slices.Contains(Ancestors(o.N), b.N) })
	})
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
	// the site's own add button; a link that adds to a cart is another
	// site's (an affiliate review's "Add to basket at Amazon")
	for _, b := range QueryAll(p.Doc, "button, input[type=submit], input[type=button]") {
		name, _ := Attr(b, "aria-label")
		if strings.TrimSpace(name) == "" {
			name = HTMLText(b)
		}
		if strings.TrimSpace(name) == "" {
			name, _ = Attr(b, "value")
		}
		if addButton.MatchString(name) {
			return true
		}
	}
	return false
}

// rowKey reports a short text that keys a row of a list or table, not a
// heading over a box: a spec's name ("Recommended For Gaming" before "Yes").
func rowKey(n *html.Node) bool {
	if n.DataAtom != atom.Span && n.DataAtom != atom.P {
		return false
	}
	for e, up := n.Parent, 0; e != nil && up < 2; e, up = e.Parent, up+1 {
		switch e.DataAtom {
		case atom.Li, atom.Td, atom.Th, atom.Dt, atom.Dd:
			return true
		}
	}
	return false
}

var (
	checkoutPath = regexp.MustCompile(`/checkouts?$`)
	// addButton is the name of a button that adds to the cart: "Add to
	// trolley", "Add item Trophy Basmati Rice 10kg to cart".
	addButton = regexp.MustCompile(`(?i)^\s*add\b.{0,120}?\bto\s+(?:the\s+|your\s+|my\s+)?(?:cart|basket|bag|trolley)\s*$`)
	// cartWord is a word that names the cart; controlFiller the words a cart
	// button's name wraps it in ("Open cart", "Basket quantity 0", "Shopping
	// Cart Menu 0 items in the cart").
	cartWord      = regexp.MustCompile(`^(?:cart|basket|bag|trolley|minicart|minibasket|minibag)$`)
	controlFiller = regexp.MustCompile(`^(?:\d+|open|view|show|go|goto|to|your|my|the|shopping|mini|menu|icon|link|quantity|click|items?|with|in|drawer|button|and|of|total)$`)
)

// CartControl reports a link or button for the cart: its name (aria-label,
// else its text with each icon's name, else its title) is the cart's, in a
// cart button's words. A
// link named "Bag Straps" sells bags; one named "Bag" opens the cart.
func CartControl(n *html.Node) bool {
	name, _ := Attr(n, "aria-label")
	if strings.TrimSpace(name) == "" {
		name = iconText(n)
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
