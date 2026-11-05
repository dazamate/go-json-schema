package vocabtool

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// The shop checks read an online shop's page for what its structured data
// should hold. They work out what kind of page it is (a product, a listing, a
// policy, a contact page), then name the nodes it calls for: the sitewide
// WebSite and business, the page's own WebPage type, and what a product page
// and the shop's policies show. Each message names the schema.org types to
// use, so a draft with nothing in it gets the whole list.

// kindPageType is the WebPage type each kind of page takes.
var kindPageType = map[webpage.Kind]string{
	webpage.KindProduct: "ItemPage", webpage.KindListing: "CollectionPage", webpage.KindSearch: "SearchResultsPage",
	webpage.KindContact: "ContactPage", webpage.KindAbout: "AboutPage", webpage.KindFAQ: "FAQPage", webpage.KindCheckout: "CheckoutPage",
}

// pageType warns when the page has no WebPage node, or one typed more
// generally than the kind of page it is.
func (c *checker) pageType(all []jsonld.Node, p *webpage.Facts) {
	want := kindPageType[p.Kind]
	if want == "" {
		want = "WebPage"
	}
	i := slices.IndexFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "WebPage") && len(l.N.Keys) > 1 })
	if i < 0 {
		extra := map[webpage.Kind]string{
			webpage.KindProduct: "mainEntity the Product, breadcrumb the BreadcrumbList",
			webpage.KindListing: "mainEntity the ItemList of its products, breadcrumb the BreadcrumbList",
			webpage.KindHome:    "about and mainEntity the business",
			webpage.KindContact: "mainEntity the business",
			webpage.KindAbout:   "mainEntity the business",
		}[p.Kind]
		if extra == "" {
			extra = "breadcrumb the BreadcrumbList when it has one"
		}
		c.add(pagePath, Warning, "the page has no WebPage: add a %s for it (url, name, isPartOf the WebSite, %s)", want, extra)
		return
	}
	l := all[i]
	ty := c.single(l.N)
	// the name and URL check may already have named the subtype
	named := slices.ContainsFunc(c.issues, func(i Issue) bool { return i.Path == l.Path && strings.Contains(i.Message, "subtype "+want) })
	if ty != "" && ty != want && slices.Contains(c.v.V.Ancestors(want), ty) && !c.suggested[l.N] && !named {
		c.add(l.Path, Warning, "this %s is a %s page: type it %s", ty, strings.TrimPrefix(string(p.Kind), "kind"), want)
		c.suggested[l.N] = true
	}
}

// website warns when the draft has no WebSite, or one without the page's
// language.
func (c *checker) website(all []jsonld.Node, p *webpage.Facts) {
	lang := ""
	if p.Doc != nil {
		for _, h := range webpage.QueryAll(p.Doc, "html[lang]") {
			lang, _ = webpage.Attr(h, "lang")
		}
	}
	i := slices.IndexFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "WebSite") && len(l.N.Keys) > 1 })
	if i < 0 {
		home, id := "the site's home page", "the home page's URL + #website"
		if p.URL != nil {
			home = p.URL.Scheme + "://" + p.URL.Host + "/"
			id = home + "#website"
		}
		c.add(pagePath, Warning, "the draft has no WebSite: add the site's WebSite to every page (@id %s, url %s, name, inLanguage %s, publisher the business) and set the WebPage's isPartOf to it", id, home, txt.OrDefault(lang, "the page's language"))
		return
	}
	if _, ok := all[i].N.Get("inLanguage"); !ok && lang != "" {
		c.add(all[i].Path, Warning, "this WebSite has no inLanguage: set it %s, the page's language", lang)
	}
	wp, ok := c.mainWebPage(all, p)
	if !ok {
		return
	}
	if _, ok := wp.N.Get("isPartOf"); !ok {
		ref := strings.Join(append(jsonld.Strs(all[i].N, "@id"), jsonld.Strs(all[i].N, "url")...), ", ")
		c.add(wp.Path, Warning, "the WebPage has no isPartOf: set it to the WebSite ({\"@id\": %q})", txt.OrDefault(strings.SplitN(ref, ", ", 2)[0], "the WebSite's @id"))
	}
	// the breadcrumb nav's BreadcrumbList belongs to the page
	crumbs := p.Doc != nil && slices.ContainsFunc(webpage.PageSections(p.Doc), func(s *webpage.Section) bool { return s.Type == "BreadcrumbList" })
	if _, ok := wp.N.Get("breadcrumb"); !ok && crumbs && c.has(all, "BreadcrumbList") {
		c.add(wp.Path, Warning, "the WebPage has no breadcrumb: set it to the BreadcrumbList of the page's breadcrumb nav")
	}
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
func businessName(p *webpage.Facts) string {
	var parts []string
	if p.Doc != nil {
		parts = append(parts, webpage.MetaContent(p.Doc, "property", "og:site_name"))
		for _, t := range webpage.QueryAll(p.Doc, "title") {
			parts = append(parts, webpage.HTMLText(t))
		}
	}
	if m := copyrightOwner.FindStringSubmatch(p.Text); m != nil {
		parts = append(parts, m[1])
	}
	return strings.Join(parts, " · ")
}

// storeType is the Store subtype for the business's trade, or Store.
func storeType(p *webpage.Facts) string {
	name := businessName(p) + " · " + p.H1()
	for _, k := range storeKinds {
		if k.words.MatchString(name) {
			return k.use
		}
	}
	return "Store"
}

// business warns when the draft has no business node, or one typed more
// generally than a shop: an online shop is an OnlineStore, and one with a
// street address the Store subtype for its trade.
func (c *checker) business(all []jsonld.Node, p *webpage.Facts) {
	addr := p.Address()
	want, why := "", ""
	switch {
	case addr != "":
		want = storeType(p)
		why = fmt.Sprintf("the page shows the street address %q", txt.Clip(addr, 60))
	case p.Sells():
		want, why = "OnlineStore", "the site sells online"
	}
	i := slices.IndexFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "Organization") && len(l.N.Keys) > 1 })
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
	ty := c.single(l.N)
	if want != "" && ty != "" && ty != want && slices.Contains(c.v.V.Ancestors(want), ty) && !c.suggested[l.N] {
		c.add(l.Path, Warning, "%s: type this %s %s", why, ty, want)
		c.suggested[l.N] = true
	}
}

// logo warns when the site header shows the business's logo and no node has
// a logo.
func (c *checker) logo(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil || slices.ContainsFunc(all, func(l jsonld.Node) bool { _, ok := l.N.Get("logo"); return ok }) {
		return
	}
	for _, s := range webpage.PageSections(p.Doc) {
		if s.Type != "WPHeader" {
			continue
		}
		for _, a := range webpage.QueryAll(s.N, "a[href]") {
			if p.HomeLink(a) && len(webpage.QueryAll(a, "img, svg")) > 0 {
				c.add("html:"+webpage.SelectorFor(p.Doc, a), Warning, "the site header shows the business's logo: set logo on the business, an ImageObject (url, width, height)")
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
func (c *checker) social(p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	var missing []string
	for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
		h, _ := webpage.Attr(a, "href")
		if !socialProfile(h) || !webpage.Shown(a) {
			continue
		}
		if k := txt.NormLink(h); !d.URLs[k] && !slices.Contains(missing, k) {
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
func (c *checker) catalog(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil || slices.ContainsFunc(all, func(l jsonld.Node) bool { _, ok := l.N.Get("hasOfferCatalog"); return ok }) {
		return
	}
	var names []string
	var seen []string
	for _, nav := range webpage.QueryAll(p.Doc, "nav, header") {
		for _, a := range webpage.QueryAll(nav, "a[href]") {
			u, ok := p.SiteLink(a)
			if !ok || slices.Contains(seen, u) {
				continue
			}
			pu, _ := url.Parse(u)
			segs := webpage.Segments(pu)
			if i := slices.IndexFunc(segs, categorySegment.MatchString); i >= 0 && i < len(segs)-1 && pu.RawQuery == "" {
				seen = append(seen, u)
				names = append(names, strings.TrimSpace(webpage.HTMLText(a)))
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
