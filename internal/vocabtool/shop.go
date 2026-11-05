package vocabtool

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The shop checks read an online shop's page for what its structured data
// should hold. They work out what kind of page it is (a product, a listing, a
// policy, a contact page), then name the nodes it calls for: the sitewide
// WebSite and business, the page's own WebPage type, and what a product page,
// a listing and the shop's policies show. Each message names the schema.org
// types to use, so a draft with nothing in it gets the whole list.

// pageKind is what a shop's page is for.
type pageKind string

const (
	kindOther    pageKind = ""
	kindHome     pageKind = "home"
	kindProduct  pageKind = "product"
	kindListing  pageKind = "listing"
	kindSearch   pageKind = "search"
	kindContact  pageKind = "contact"
	kindAbout    pageKind = "about"
	kindShipping pageKind = "shipping"
	kindReturns  pageKind = "returns"
	kindFAQ      pageKind = "faq"
	kindCart     pageKind = "cart"
	kindCheckout pageKind = "checkout"
	kindAccount  pageKind = "account"
)

// kindPageType is the WebPage type each kind of page takes.
var kindPageType = map[pageKind]string{
	kindProduct: "ItemPage", kindListing: "CollectionPage", kindSearch: "SearchResultsPage",
	kindContact: "ContactPage", kindAbout: "AboutPage", kindFAQ: "FAQPage", kindCheckout: "CheckoutPage",
}

// kindSegments are the URL path segments that say what a page is.
var kindSegments = []struct {
	kind pageKind
	re   *regexp.Regexp
}{
	{kindCart, regexp.MustCompile(`^(cart|basket|bag|shopping-bag|shopping-cart)$`)},
	{kindCheckout, regexp.MustCompile(`^checkouts?$`)},
	{kindAccount, regexp.MustCompile(`^(account|my-account|login|signin|sign-in|register|signup|sign-up|wishlist|wish-list|favourites|favorites)$`)},
	{kindSearch, regexp.MustCompile(`^search$`)},
	{kindContact, regexp.MustCompile(`^contact(-us)?$`)},
	{kindAbout, regexp.MustCompile(`^(about|about-us|our-story)$`)},
	{kindShipping, regexp.MustCompile(`^(shipping|delivery|postage)(-(policy|information|info))?$`)},
	{kindReturns, regexp.MustCompile(`^(returns?|refunds?|exchanges?)(-(policy|information|info))?$|^(return|refund)-policy$`)},
	{kindFAQ, regexp.MustCompile(`^(faqs?|frequently-asked-questions)$`)},
}

var (
	productSegment = regexp.MustCompile(`^(products?|p|item|items)$`)
	listingSegment = regexp.MustCompile(`^(categor(y|ies)|collections?|c|shop|store|catalog(ue)?|departments?)$`)
	localeSegment  = regexp.MustCompile(`^[a-z]{2}(-[a-z]{2})?$`)
	kindHeadings   = []struct {
		kind pageKind
		re   *regexp.Regexp
	}{
		{kindSearch, regexp.MustCompile(`(?i)^search results|^results for`)},
		{kindCart, regexp.MustCompile(`(?i)^your (bag|cart|basket)\b`)},
		{kindCheckout, regexp.MustCompile(`(?i)^checkout\b`)},
		{kindContact, regexp.MustCompile(`(?i)^contact\b`)},
		{kindAbout, regexp.MustCompile(`(?i)^about\b|^our story\b`)},
		{kindShipping, regexp.MustCompile(`(?i)^(shipping|delivery|postage)\b`)},
		{kindReturns, regexp.MustCompile(`(?i)^(returns?|refunds?)\b`)},
		{kindFAQ, regexp.MustCompile(`(?i)^(faqs?|frequently asked)\b`)},
	}
	cartPath  = regexp.MustCompile(`/(cart|basket|bag)$`)
	addToCart = regexp.MustCompile(`(?i)\b(add to (cart|bag|basket)|buy now|pre-?order)\b`)
)

// segments are a URL path's segments, lowercased.
func segments(u *url.URL) []string {
	if u == nil {
		return nil
	}
	return strings.FieldsFunc(strings.ToLower(u.Path), func(r rune) bool { return r == '/' })
}

// h1 is the text of the page's first visible h1.
func (p *pageFacts) h1() string {
	if p.doc == nil {
		return ""
	}
	var h string
	visit(p.doc, func(n *xhtml.Node) {
		if h == "" && n.DataAtom == atom.H1 {
			h = htmlText(n)
		}
	})
	return h
}

// metaContent is the content of the first meta element with the attribute.
func metaContent(doc *xhtml.Node, key, name string) string {
	for _, n := range queryAll(doc, "meta") {
		if v, _ := attr(n, key); strings.EqualFold(v, name) {
			c, _ := attr(n, "content")
			return strings.TrimSpace(c)
		}
	}
	return ""
}

// noindex reports a robots meta tag that keeps the page out of search.
func noindex(doc *xhtml.Node) bool {
	for _, name := range []string{"robots", "googlebot"} {
		if strings.Contains(strings.ToLower(metaContent(doc, "name", name)), "noindex") {
			return true
		}
	}
	return false
}

// buyForm is the page's add-to-cart form outside product cards, or nil.
func (p *pageFacts) buyForm() *xhtml.Node {
	if p.doc == nil {
		return nil
	}
	var form *xhtml.Node
	visit(p.doc, func(n *xhtml.Node) {
		if form != nil || n.DataAtom != atom.Form || p.inCard(n) {
			return
		}
		action, _ := attr(n, "action")
		if cartForm.MatchString(action) || addToCart.MatchString(htmlText(n)) {
			form = n
		}
	})
	return form
}

// classify works out what kind of page this is: from its address, then its
// og:type, add-to-cart form and product cards, then its h1.
func (p *pageFacts) classify() pageKind {
	segs := segments(p.url)
	for _, ks := range kindSegments {
		if slices.ContainsFunc(segs, ks.re.MatchString) {
			return ks.kind
		}
	}
	if p.doc != nil && strings.HasPrefix(strings.ToLower(metaContent(p.doc, "property", "og:type")), "product") {
		return kindProduct
	}
	if i := slices.IndexFunc(segs, productSegment.MatchString); i >= 0 && i < len(segs)-1 {
		return kindProduct
	}
	h := p.h1()
	for _, kh := range kindHeadings {
		if kh.re.MatchString(h) {
			return kh.kind
		}
	}
	if p.buyForm() != nil && h != "" {
		return kindProduct
	}
	if len(segs) == 0 || (len(segs) == 1 && localeSegment.MatchString(segs[0])) {
		if p.url != nil {
			return kindHome
		}
	}
	listed := 0
	for _, c := range p.productCards() {
		if !c.related {
			listed++
		}
	}
	if (slices.ContainsFunc(segs, listingSegment.MatchString) && listed >= 2) || listed >= 3 {
		return kindListing
	}
	return kindOther
}

// privateKinds are pages search engines don't index, so the shop checks
// leave them be.
var privateKinds = map[pageKind]string{kindCart: "a shopping cart", kindCheckout: "a checkout", kindAccount: "an account page"}

// card is a product card on a listing: the element that holds one product's
// link and price.
type card struct {
	n       *xhtml.Node
	url     string
	name    string
	related bool // in a block the page labels as related products
}

// siteLink is a link's address resolved and normalized when it points at
// another page of this site.
func (p *pageFacts) siteLink(a *xhtml.Node) (string, bool) {
	h, _ := attr(a, "href")
	h = strings.TrimSpace(h)
	if h == "" || strings.HasPrefix(h, "#") {
		return "", false
	}
	u, err := url.Parse(h)
	if err != nil || (u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	if p.base != nil {
		u = p.base.ResolveReference(u)
		if !strings.EqualFold(u.Hostname(), p.base.Hostname()) {
			return "", false
		}
	} else if u.IsAbs() {
		return "", false
	}
	s := normURL(u)
	if p.url != nil && s == normURL(p.url) {
		return "", false
	}
	return s, true
}

// siteLinks are the distinct pages of this site that links under n point at.
func (p *pageFacts) siteLinks(n *xhtml.Node) []string {
	var out []string
	visit(n, func(a *xhtml.Node) {
		if a.DataAtom == atom.A {
			if s, ok := p.siteLink(a); ok && !slices.Contains(out, s) {
				out = append(out, s)
			}
		}
	})
	return out
}

func holds(n *xhtml.Node, a atom.Atom) bool {
	found := false
	visit(n, func(c *xhtml.Node) { found = found || c.DataAtom == a })
	return found
}

// productCards finds the page's product cards once: for each link to
// another page of the site, outside its header, footer and navs, the nearest
// element around it that shows a price and links to no other page. An
// element holding an h1 or a form is the page's own product, not a card. A
// single priced link is no listing, so it takes two.
func (p *pageFacts) productCards() []*card {
	if p.cardsDone {
		return p.cards
	}
	p.cardsDone = true
	if p.doc == nil {
		return nil
	}
	seen := map[*xhtml.Node]bool{}
	var out []*card
	visit(p.doc, func(a *xhtml.Node) {
		if a.DataAtom != atom.A || chrome(a) {
			return
		}
		u, ok := p.siteLink(a)
		if !ok {
			return
		}
		for e, depth := a.Parent, 0; e != nil && depth < 8; e, depth = e.Parent, depth+1 {
			if e.DataAtom == atom.Body || e.DataAtom == atom.Main || holds(e, atom.H1) || holds(e, atom.Form) {
				return
			}
			if len(p.siteLinks(e)) > 1 {
				return
			}
			if priceAmount.MatchString(htmlText(e)) {
				if !seen[e] {
					seen[e] = true
					out = append(out, &card{n: e, url: u, name: cardName(e, a)})
				}
				return
			}
		}
	})
	if len(out) < 2 {
		return nil
	}
	for _, b := range p.relatedBoxes() {
		for _, c := range out {
			for e := c.n; e != nil; e = e.Parent {
				if e == b.n {
					c.related = true
				}
			}
		}
	}
	p.cards = out
	return out
}

// cardName is a card's heading, else its link's text, image alt or label.
func cardName(c, a *xhtml.Node) string {
	name := ""
	visit(c, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			if name == "" {
				name = htmlText(n)
			}
		}
	})
	if name == "" {
		name = htmlText(a)
	}
	if name == "" {
		for _, img := range queryAll(a, "img[alt]") {
			name, _ = attr(img, "alt")
		}
	}
	if name == "" {
		name, _ = attr(a, "aria-label")
	}
	// a link that wraps the whole card carries its price too
	return strings.Join(strings.Fields(priceAmount.ReplaceAllString(name, "")), " ")
}

// inCard reports whether n sits in a product card.
func (p *pageFacts) inCard(n *xhtml.Node) bool {
	if !p.cardsDone {
		return false
	}
	for _, c := range p.cards {
		for e := n; e != nil; e = e.Parent {
			if e == c.n {
				return true
			}
		}
	}
	return false
}

// relatedBox is a block the page labels as related reading or products.
type relatedBox struct {
	label string
	n     *xhtml.Node
}

// relatedBoxes finds the blocks a short label ("You may also like", "Keep
// reading") heads: the nearest section, aside, nav or article around the
// label, else the nearest element around it that links to other pages.
func (p *pageFacts) relatedBoxes() []relatedBox {
	if p.doc == nil {
		return nil
	}
	var out []relatedBox
	visit(p.doc, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.P, atom.Span:
		default:
			return
		}
		t := strings.TrimSpace(htmlText(n))
		// a guide's own nav of its pages sits in the content, not the site's
		// chrome
		if len(strings.Fields(t)) > 6 || !relatedLabel.MatchString(t) || (chrome(n) && !(guideLabel.MatchString(t) && !siteChromeAround(n))) {
			return
		}
		for e := n.Parent; e != nil && e.DataAtom != atom.Body && e.DataAtom != atom.Main; e = e.Parent {
			switch e.DataAtom {
			case atom.Section, atom.Aside, atom.Nav, atom.Article:
			default:
				if len(p.siteLinks(e)) == 0 {
					continue
				}
			}
			if !slices.ContainsFunc(out, func(b relatedBox) bool { return b.n == e }) {
				out = append(out, relatedBox{t, e})
			}
			return
		}
	})
	return out
}

// sells reports a page of a shop: a link to its cart, an add-to-cart form or
// product cards.
func (p *pageFacts) sells() bool {
	if p.doc == nil {
		return false
	}
	if p.buyForm() != nil || len(p.productCards()) > 0 {
		return true
	}
	for _, a := range queryAll(p.doc, "a[href]") {
		if u, ok := p.siteLink(a); ok && cartPath.MatchString(u) {
			return true
		}
	}
	return false
}

// promo reports a shop's dl that sells rather than defines: one whose block
// links into a category or collection ("Shop lockets").
func (p *pageFacts) promo(dl *xhtml.Node) bool {
	if !p.sells() {
		return false
	}
	for e, up := dl.Parent, 0; e != nil && up < 2 && e.DataAtom != atom.Main && e.DataAtom != atom.Body; e, up = e.Parent, up+1 {
		for _, a := range queryAll(e, "a[href]") {
			if u, ok := p.siteLink(a); ok && listingLink(mustParse(u)) {
				return true
			}
		}
	}
	return false
}

// listingLink reports an address of a category or collection: its first
// segment after any locale names one (/au/collections/lockets).
func listingLink(u *url.URL) bool {
	segs := segments(u)
	if len(segs) > 0 && localeSegment.MatchString(segs[0]) {
		segs = segs[1:]
	}
	return len(segs) > 1 && listingSegment.MatchString(segs[0])
}

// homePath reports a link to the site's home page: its root, or a locale root
// such as /au.
func (p *pageFacts) homeLink(a *xhtml.Node) bool {
	h, _ := attr(a, "href")
	u, err := url.Parse(strings.TrimSpace(h))
	if err != nil || h == "" {
		return false
	}
	if p.base != nil {
		u = p.base.ResolveReference(u)
		if !strings.EqualFold(u.Hostname(), p.base.Hostname()) {
			return false
		}
	}
	segs := segments(u)
	return len(segs) == 0 || (len(segs) == 1 && localeSegment.MatchString(segs[0]))
}

// pageType warns when the page has no WebPage node, or one typed more
// generally than the kind of page it is.
func (c *checker) pageType(all []located, p *pageFacts) {
	want := kindPageType[p.kind]
	if want == "" {
		want = "WebPage"
	}
	i := slices.IndexFunc(all, func(l located) bool { return c.is(l.n, "WebPage") && len(l.n.keys) > 1 })
	if i < 0 {
		extra := map[pageKind]string{
			kindProduct: "mainEntity the Product, breadcrumb the BreadcrumbList",
			kindListing: "mainEntity the ItemList of its products, breadcrumb the BreadcrumbList",
			kindHome:    "about and mainEntity the business",
			kindContact: "mainEntity the business",
			kindAbout:   "mainEntity the business",
		}[p.kind]
		if extra == "" {
			extra = "breadcrumb the BreadcrumbList when it has one"
		}
		c.add(pagePath, Warning, "the page has no WebPage: add a %s for it (url, name, isPartOf the WebSite, %s)", want, extra)
		return
	}
	l := all[i]
	ty := c.single(l.n)
	// the name and URL check may already have named the subtype
	named := slices.ContainsFunc(c.issues, func(i Issue) bool { return i.Path == l.path && strings.Contains(i.Message, "subtype "+want) })
	if ty != "" && ty != want && slices.Contains(c.v.V.Ancestors(want), ty) && !c.suggested[l.n] && !named {
		c.add(l.path, Warning, "this %s is a %s page: type it %s", ty, strings.TrimPrefix(string(p.kind), "kind"), want)
		c.suggested[l.n] = true
	}
}

// website warns when the draft has no WebSite, or one without the page's
// language.
func (c *checker) website(all []located, p *pageFacts) {
	lang := ""
	if p.doc != nil {
		for _, h := range queryAll(p.doc, "html[lang]") {
			lang, _ = attr(h, "lang")
		}
	}
	i := slices.IndexFunc(all, func(l located) bool { return c.is(l.n, "WebSite") && len(l.n.keys) > 1 })
	if i < 0 {
		home, id := "the site's home page", "the home page's URL + #website"
		if p.url != nil {
			home = p.url.Scheme + "://" + p.url.Host + "/"
			id = home + "#website"
		}
		c.add(pagePath, Warning, "the draft has no WebSite: add the site's WebSite to every page (@id %s, url %s, name, inLanguage %s, publisher the business) and set the WebPage's isPartOf to it", id, home, orDefault(lang, "the page's language"))
		return
	}
	if _, ok := all[i].n.get("inLanguage"); !ok && lang != "" {
		c.add(all[i].path, Warning, "this WebSite has no inLanguage: set it %s, the page's language", lang)
	}
	wp, ok := c.mainWebPage(all, p)
	if !ok {
		return
	}
	if _, ok := wp.n.get("isPartOf"); !ok {
		ref := strings.Join(append(strs(all[i].n, "@id"), strs(all[i].n, "url")...), ", ")
		c.add(wp.path, Warning, "the WebPage has no isPartOf: set it to the WebSite ({\"@id\": %q})", orDefault(strings.SplitN(ref, ", ", 2)[0], "the WebSite's @id"))
	}
	// the breadcrumb nav's BreadcrumbList belongs to the page
	crumbs := p.doc != nil && slices.ContainsFunc(pageSections(p.doc), func(s *section) bool { return s.Type == "BreadcrumbList" })
	if _, ok := wp.n.get("breadcrumb"); !ok && crumbs && c.has(all, "BreadcrumbList") {
		c.add(wp.path, Warning, "the WebPage has no breadcrumb: set it to the BreadcrumbList of the page's breadcrumb nav")
	}
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// storeKinds name the Store subtype a shopfront business takes from the words
// its name uses, most specific first.
var storeKinds = []struct {
	words *regexp.Regexp
	use   string
}{
	{regexp.MustCompile(`(?i)\b(menswear|men'?s (clothing|wear|fashion))\b`), "MensClothingStore"},
	{regexp.MustCompile(`(?i)\b(jewell?e?ry|jewell?ers?)\b`), "JewelryStore"},
	{regexp.MustCompile(`(?i)\b(auto|car|motor) parts\b`), "AutoPartsStore"},
	{regexp.MustCompile(`(?i)\b(bikes?|bicycles?|cycles?)\b`), "BikeStore"},
	{regexp.MustCompile(`(?i)\b(books?|bookshop|bookstore|booksellers?)\b`), "BookStore"},
	{regexp.MustCompile(`(?i)\b(computers?|laptops?)\b`), "ComputerStore"},
	{regexp.MustCompile(`(?i)\bconvenience\b`), "ConvenienceStore"},
	{regexp.MustCompile(`(?i)\bdepartment stores?\b`), "DepartmentStore"},
	{regexp.MustCompile(`(?i)\belectronics?\b`), "ElectronicsStore"},
	{regexp.MustCompile(`(?i)\b(florists?|flowers)\b`), "Florist"},
	{regexp.MustCompile(`(?i)\bfurniture\b`), "FurnitureStore"},
	{regexp.MustCompile(`(?i)\b(garden|nursery)\b`), "GardenStore"},
	{regexp.MustCompile(`(?i)\b(grocery|grocers?|supermarket)\b`), "GroceryStore"},
	{regexp.MustCompile(`(?i)\bhardware\b`), "HardwareStore"},
	{regexp.MustCompile(`(?i)\b(hobby|hobbies|crafts?)\b`), "HobbyShop"},
	{regexp.MustCompile(`(?i)\b(homewares?|home goods)\b`), "HomeGoodsStore"},
	{regexp.MustCompile(`(?i)\b(liquor|bottle shop|wine shop)\b`), "LiquorStore"},
	{regexp.MustCompile(`(?i)\b(mobile phones?|phones)\b`), "MobilePhoneStore"},
	{regexp.MustCompile(`(?i)\b(movie|video) rentals?\b`), "MovieRentalStore"},
	{regexp.MustCompile(`(?i)\b(music|records|instruments)\b`), "MusicStore"},
	{regexp.MustCompile(`(?i)\boffice (supplies|equipment)\b|\bstationery\b`), "OfficeEquipmentStore"},
	{regexp.MustCompile(`(?i)\boutlet\b`), "OutletStore"},
	{regexp.MustCompile(`(?i)\bpawn(brokers?|shop)?\b`), "PawnShop"},
	{regexp.MustCompile(`(?i)\bpets?\b`), "PetStore"},
	{regexp.MustCompile(`(?i)\b(shoes|footwear)\b`), "ShoeStore"},
	{regexp.MustCompile(`(?i)\b(sports?|sporting goods)\b`), "SportingGoodsStore"},
	{regexp.MustCompile(`(?i)\b(tyres?|tires?)\b`), "TireShop"},
	{regexp.MustCompile(`(?i)\btoys?\b`), "ToyStore"},
	{regexp.MustCompile(`(?i)\bwholesale\b`), "WholesaleStore"},
	{regexp.MustCompile(`(?i)\b(clothing|fashion|apparel|boutique)\b`), "ClothingStore"},
}

var copyrightOwner = regexp.MustCompile(`©\s*(?:\d{4}\s*[-–]?\s*)*([^·|\n]{2,60})`)

// businessName is what the page calls the business: its og:site_name, its
// copyright holder, and its title.
func (p *pageFacts) businessName() string {
	var parts []string
	if p.doc != nil {
		parts = append(parts, metaContent(p.doc, "property", "og:site_name"))
		for _, t := range queryAll(p.doc, "title") {
			parts = append(parts, htmlText(t))
		}
	}
	if m := copyrightOwner.FindStringSubmatch(p.text); m != nil {
		parts = append(parts, m[1])
	}
	return strings.Join(parts, " · ")
}

// storeType is the Store subtype for the business's trade, or Store.
func (p *pageFacts) storeType() string {
	name := p.businessName() + " · " + p.h1()
	for _, k := range storeKinds {
		if k.words.MatchString(name) {
			return k.use
		}
	}
	return "Store"
}

var streetAddress = regexp.MustCompile(`\b\d{1,5}[A-Za-z]?\s+(?:[A-Z][a-z]+\s){1,3}(?:Street|St|Road|Rd|Avenue|Ave|Lane|Ln|Drive|Dr|Boulevard|Blvd|Highway|Hwy|Parade|Pde|Place|Pl|Terrace|Tce|Way|Court|Ct|Crescent|Cres)\b\.?[^.\n]{0,60}`)

// address is a street address the page shows: an address element, or a
// number and street name.
func (p *pageFacts) address() string {
	if p.doc != nil {
		for _, n := range queryAll(p.doc, "address") {
			if t := htmlText(n); t != "" && shown(n) {
				return t
			}
		}
	}
	return strings.TrimSpace(streetAddress.FindString(p.main))
}

// business warns when the draft has no business node, or one typed more
// generally than a shop: an online shop is an OnlineStore, and one with a
// street address the Store subtype for its trade.
func (c *checker) business(all []located, p *pageFacts) {
	addr := p.address()
	want, why := "", ""
	switch {
	case addr != "":
		want = p.storeType()
		why = fmt.Sprintf("the page shows the street address %q", clip(addr, 60))
	case p.sells():
		want, why = "OnlineStore", "the site sells online"
	}
	i := slices.IndexFunc(all, func(l located) bool { return c.is(l.n, "Organization") && len(l.n.keys) > 1 })
	if i < 0 {
		switch want {
		case "":
			c.add(pagePath, Warning, "the page has no business node: add the Organization that runs the site (name, url, logo, sameAs) as the WebSite's publisher")
		case "OnlineStore":
			c.add(pagePath, Warning, "the site sells online but the page has no business node: add an OnlineStore (the OnlineBusiness subtype of Organization for a shop that trades online) as the WebSite's publisher, with name, url, logo, sameAs, email, taxID, hasMerchantReturnPolicy and hasShippingService")
		default:
			c.add(pagePath, Warning, "%s but no business node: add the business as a %s (the Store subtype for its trade, a LocalBusiness: name, address, geo, openingHoursSpecification, telephone)", why, want)
		}
		return
	}
	l := all[i]
	ty := c.single(l.n)
	if want != "" && ty != "" && ty != want && slices.Contains(c.v.V.Ancestors(want), ty) && !c.suggested[l.n] {
		c.add(l.path, Warning, "%s: type this %s %s", why, ty, want)
		c.suggested[l.n] = true
	}
}

// logo warns when the site header shows the business's logo and no node has
// a logo.
func (c *checker) logo(all []located, p *pageFacts) {
	if p.doc == nil || slices.ContainsFunc(all, func(l located) bool { _, ok := l.n.get("logo"); return ok }) {
		return
	}
	for _, s := range pageSections(p.doc) {
		if s.Type != "WPHeader" {
			continue
		}
		for _, a := range queryAll(s.n, "a[href]") {
			if p.homeLink(a) && len(queryAll(a, "img, svg")) > 0 {
				c.add("html:"+selectorFor(p.doc, a), Warning, "the site header shows the business's logo: set logo on the business, an ImageObject (url, width, height)")
				return
			}
		}
	}
}

var (
	socialHost  = regexp.MustCompile(`(?i)^(?:www\.|m\.)?(instagram\.com|facebook\.com|fb\.com|tiktok\.com|pinterest\.[a-z.]+|youtube\.com|x\.com|twitter\.com|linkedin\.com|threads\.net|etsy\.com|snapchat\.com|bsky\.app)$`)
	shareLink   = regexp.MustCompile(`(?i)facebook\.com/sharer|(?:twitter|x)\.com/(?:intent|share)|pinterest\.[a-z.]+/pin/create|linkedin\.com/(?:sharing|shareArticle)|wa\.me/\?|api\.whatsapp\.com/send|reddit\.com/submit`)
	socialSkips = regexp.MustCompile(`(?i)/(?:sharer|intent|share|pin/create|sharing|dialog|plugins)\b|privacy|polic(?:y|ies)|terms|legal|/(?:help|about|settings|login|tos)(?:/|$)`)
)

// socialProfile is a link to one of the business's social profiles.
func socialProfile(href string) bool {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil || !u.IsAbs() || !socialHost.MatchString(u.Hostname()) || socialSkips.MatchString(u.Path) {
		return false
	}
	return strings.Trim(u.Path, "/") != ""
}

// social warns on links to the business's social profiles that no sameAs
// lists.
func (c *checker) social(p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	var missing []string
	for _, a := range queryAll(p.doc, "a[href]") {
		h, _ := attr(a, "href")
		if !socialProfile(h) || !shown(a) {
			continue
		}
		if k := normLink(h); !d.urls[k] && !slices.Contains(missing, k) {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		short := make([]string, len(missing))
		for i, m := range missing {
			short[i] = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(m, "https://"), "http://"), "www.")
		}
		c.add(pagePath, Warning, "the page links the business's profiles (%s) that no sameAs lists: add them to the business's sameAs, and a FollowAction for each (target the profile) to its potentialAction", strings.Join(short, ", "))
	}
}

var categorySegment = regexp.MustCompile(`^(categor(y|ies)|collections?|c|departments?)$`)

// catalog warns when the site's menus list its categories and no node has an
// OfferCatalog of them.
func (c *checker) catalog(all []located, p *pageFacts) {
	if p.doc == nil || slices.ContainsFunc(all, func(l located) bool { _, ok := l.n.get("hasOfferCatalog"); return ok }) {
		return
	}
	var names []string
	var seen []string
	for _, nav := range queryAll(p.doc, "nav, header") {
		for _, a := range queryAll(nav, "a[href]") {
			u, ok := p.siteLink(a)
			if !ok || slices.Contains(seen, u) {
				continue
			}
			pu, _ := url.Parse(u)
			segs := segments(pu)
			if i := slices.IndexFunc(segs, categorySegment.MatchString); i >= 0 && i < len(segs)-1 && pu.RawQuery == "" {
				seen = append(seen, u)
				names = append(names, strings.TrimSpace(htmlText(a)))
			}
		}
	}
	if len(seen) >= 3 {
		list := names
		if len(list) > 6 {
			list = append(list[:6:6], "…")
		}
		c.add(pagePath, Warning, "the site's menu lists %d categories (%s): set the business's hasOfferCatalog, an OfferCatalog with a ListItem for each category naming its CollectionPage", len(seen), strings.Join(list, ", "))
	}
}

// listing warns on the product cards a page shows that no ItemList carries,
// on a product count no ItemList has as numberOfItems, and on a sort order
// no ItemList has as itemListOrder.
func (c *checker) listing(all []located, p *pageFacts, d *draftIndex) {
	// a filter link (Under $10, $20 and over) names no item's price: its
	// address holds a query, and its text is little besides the price
	if p.doc != nil {
		for _, a := range queryAll(p.doc, "a[href]") {
			if u, ok := p.siteLink(a); ok && mustParse(u).RawQuery != "" {
				if t := htmlText(a); len(strings.Fields(priceAmount.ReplaceAllString(t, ""))) <= 3 {
					p.claim(t)
				}
			}
		}
	}
	var listed []*card
	for _, cd := range p.productCards() {
		if !cd.related {
			listed = append(listed, cd)
		}
		p.claim(htmlText(cd.n))
	}
	if len(listed) < 2 {
		return
	}
	var missing []string
	for _, cd := range listed {
		if !d.urls[cd.url] && !slices.Contains(missing, cd.name) {
			missing = append(missing, cd.name)
		}
	}
	// a listing's products are its subject; a row of them elsewhere (the
	// home page's best sellers) is a part of the page
	where := "the WebPage's hasPart (name its heading)"
	if p.kind == kindListing {
		where = "the CollectionPage's mainEntity (numberOfItems, itemListOrder)"
	}
	if len(missing) > 0 {
		names := missing
		if len(names) > 5 {
			names = append(names[:5:5], "…")
		}
		c.add("html:"+selectorFor(p.doc, commonParent(listed)), Warning, "the page lists %d products (%s) that no ItemList carries: add an ItemList as %s, with a ListItem for each product (position, url) holding its Product (name, url, image, offers an Offer with price and priceCurrency)", len(missing), strings.Join(names, ", "), where)
	}
	if m := fromPrice.FindStringSubmatch(p.main); m != nil && p.kind == kindListing {
		p.claim(m[0])
		if !c.has(all, "AggregateOffer") {
			c.add(pagePath, Warning, "the page says %q: give the CollectionPage offers, an AggregateOffer (lowPrice %s, highPrice, priceCurrency, offerCount)", m[0], digitsOf(m[1]))
		}
	}
	if m := productCount.FindStringSubmatch(p.main); m != nil && p.kind == kindListing {
		n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
		if n >= len(listed) && !slices.ContainsFunc(all, func(l located) bool { return slices.Contains(scalars(l.n, "numberOfItems"), strconv.Itoa(n)) }) {
			c.add(pagePath, Warning, "the page says %q: set numberOfItems %d on the ItemList of its products", m[0], n)
		}
	}
	c.sortOrder(all, p)
}

var fromPrice = regexp.MustCompile(`(?i)\b(?:prices? (?:start|starting) (?:at|from)|starting (?:at|from)|from only)\s+((?:A\$|AU\$|\$|£|€)\s?\d+(?:\.\d{1,2})?)`)

var productCount = regexp.MustCompile(`(?i)\b(\d{1,3}(?:,\d{3})*|\d+)\s+(?:products|items|pieces|results|styles|designs)\b`)

// commonParent is the nearest element around every card.
func commonParent(cards []*card) *xhtml.Node {
	n := cards[0].n
	for n != nil {
		if !slices.ContainsFunc(cards, func(cd *card) bool {
			for e := cd.n; e != nil; e = e.Parent {
				if e == n {
					return false
				}
			}
			return true
		}) {
			return n
		}
		n = n.Parent
	}
	return cards[0].n
}

var (
	sortAsc  = regexp.MustCompile(`(?i)low to high|ascending|a\s*-\s*z|oldest`)
	sortDesc = regexp.MustCompile(`(?i)high to low|descending|z\s*-\s*a|newest`)
)

// sortOrder warns when a sort control shows an ascending or descending order
// and no ItemList has an itemListOrder.
func (c *checker) sortOrder(all []located, p *pageFacts) {
	if p.doc == nil || slices.ContainsFunc(all, func(l located) bool { _, ok := l.n.get("itemListOrder"); return ok }) {
		return
	}
	for _, sel := range queryAll(p.doc, "select") {
		name, _ := attr(sel, "name")
		id, _ := attr(sel, "id")
		if !strings.Contains(strings.ToLower(name+" "+id), "sort") {
			continue
		}
		for _, o := range queryAll(sel, "option[selected]") {
			t := htmlText(o)
			member := ""
			switch {
			case sortAsc.MatchString(t):
				member = "ItemListOrderAscending"
			case sortDesc.MatchString(t):
				member = "ItemListOrderDescending"
			}
			if member != "" {
				c.add("html:"+selectorFor(p.doc, sel), Warning, "the page sorts its products %q: set the ItemList's itemListOrder https://schema.org/%s (an ItemListOrderType)", t, member)
				return
			}
		}
	}
}

// claim notes prices another check reports, so prices does not report them
// again.
func (p *pageFacts) claim(text string) {
	for _, m := range priceAmount.FindAllStringSubmatch(text, -1) {
		if f, ok := priceOf(m); ok {
			p.claimed[f] = true
		}
	}
}

// priceOf is the amount a priceAmount match gives.
func priceOf(m []string) (float64, bool) {
	if m[2] == "" {
		return amount(m[1])
	}
	return amount(m[1] + "." + m[2])
}
