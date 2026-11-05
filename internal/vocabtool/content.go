package vocabtool

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The content checks hold the markup to the rendered page in both
// directions. Text the markup repeats must be on the page (a HowToStep, a
// Question, an Answer, a Review), and what the page shows that a type or a
// property describes must be in the markup (a copyright line, a reading time,
// images, a video, ratings, quotations, terms, counted lists, links, a
// search box, a form, a phone number, a registration number).

const pagePath = audit.PagePath // an issue about the page as a whole

// is reports whether the node has the type or a subtype of it.
func (c *checker) is(n *jsonld.Object, ty string) bool {
	return c.v.Is(jsonld.TypeNames(n), ty)
}

// pageNode is the path of the WebPage the draft describes the page with:
// the one whose url is the canonical URL, or else the first WebPage.
func (c *checker) pageNode(all []jsonld.Node, p *webpage.Facts) string {
	first := ""
	for _, l := range all {
		if !c.is(l.N, "WebPage") || len(l.N.Keys) < 2 {
			continue
		}
		if p.Base != nil && slices.ContainsFunc(jsonld.Strs(l.N, "url"), func(s string) bool { return txt.NormLink(s) == txt.NormURL(p.Base) }) {
			return l.Path
		}
		if first == "" {
			first = l.Path
		}
	}
	if first == "" {
		return "$"
	}
	return first
}

// privateNotes say what a page search engines skip would be typed as.
var privateNotes = map[webpage.Kind]string{
	webpage.KindSearch:   "its WebPage is a SearchResultsPage",
	webpage.KindCheckout: "its WebPage is a CheckoutPage",
}

// content runs the checks that compare the draft with the page. What the
// markup repeats is held to the page always; what the page shows is asked of
// the markup only on a page search engines index.
func (c *checker) content(all []jsonld.Node, p *webpage.Facts) {
	d := jsonld.NewIndex(all)
	if p.Private != "" {
		what := p.Private
		if p.Private == "noindex" {
			what = "the page is noindex"
		} else {
			what = "the page is " + what
		}
		note := ""
		if n := privateNotes[p.Kind]; n != "" {
			note = "; if you mark it up at all, " + n
		}
		c.add(pagePath, Info, "%s: search engines leave its structured data out, so the shop checks skip it%s", what, note)
		return
	}
	c.shop(all, p, d)
	c.copyright(all, p)
	c.readTime(all, p)
	c.sectionLabel(all, p)
	c.media(all, p)
	c.search(all, p)
	c.contacts(all, p)
	c.forms(all, p, d)
	c.registrations(all, p)
	c.terms(all, p)
	c.images(all, p)
	c.counts(all, p)
	c.outsideLinks(p, d)
	c.related(all, p, d)
	c.responseTime(all, p)
	c.quotes(p, d)
	c.ratings(all, p)
}

// shop runs the shop checks still here: the sitewide WebSite, logo and
// profiles, and what a product page and the shop's policies show.
func (c *checker) shop(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	if p.Doc != nil {
		c.website(all, p)
		c.logo(all, p)
		c.social(p, d)
		c.catalog(all, p)
	}
	if p.Kind == webpage.KindProduct {
		c.product(all, p, d)
	}
	// what a shop's pages state about its own sales; on another site the
	// same words ("a damaged item", "pay by bank transfer") are about others
	if p.Sells() {
		c.stock(all, p)
		c.members(all, p)
		c.returns(all, p)
		c.payments(all, p)
	}
	c.engagement(all, p)
	c.whereAndWhen(all, p)
	c.faq(all, p)
	c.founding(all, p)
}

// copyright warns when the page shows a copyright year the markup lacks.
func (c *checker) copyright(all []jsonld.Node, p *webpage.Facts) {
	m := webpage.CopyrightLine.FindStringSubmatch(p.Text)
	if m == nil {
		return
	}
	hasYear, hasHolder := false, false
	for _, l := range all {
		hasYear = hasYear || slices.Contains(jsonld.Scalars(l.N, "copyrightYear"), m[1])
		if _, ok := l.N.Get("copyrightHolder"); ok {
			hasHolder = true
		}
	}
	if !hasYear || !hasHolder {
		c.add(c.pageNode(all, p), Warning, "the page shows %q but the markup has no copyrightYear %s and copyrightHolder: set both on the WebPage or the WebSite", strings.TrimSpace(m[0]), m[1])
	}
}

var readingTime = regexp.MustCompile(`(?i)\b(\d{1,3})[\s-]*min(?:ute)?s?\.?\s+read\b`)

// readTime warns when the page gives a reading time and no node has it as
// timeRequired.
func (c *checker) readTime(all []jsonld.Node, p *webpage.Facts) {
	m := readingTime.FindStringSubmatch(p.Text)
	if m == nil {
		return
	}
	want := "PT" + m[1] + "M"
	target := ""
	for _, l := range all {
		if !c.is(l.N, "CreativeWork") {
			continue
		}
		for _, v := range jsonld.Scalars(l.N, "timeRequired") {
			if !strings.EqualFold(v, want) {
				c.add(l.Path+".timeRequired", Warning, "timeRequired is %s but the page says %q: make it %s", v, m[0], want)
			}
			return
		}
		if target == "" && c.is(l.N, "Article") {
			target = l.Path
		}
	}
	if target == "" {
		target = c.pageNode(all, p)
	}
	c.add(target, Warning, "the page says %q but no node has timeRequired: set timeRequired %s on the Article", m[0], want)
}

// sectionLabel warns when a short label sits right above the page's h1 (a
// kicker naming the section) and the Article does not carry it as its
// articleSection.
func (c *checker) sectionLabel(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil {
		return
	}
	i := slices.IndexFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "Article") })
	if i < 0 {
		return
	}
	var h1 *xhtml.Node
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if h1 == nil && n.DataAtom == atom.H1 {
			h1 = n
		}
	})
	if h1 == nil {
		return
	}
	prev := h1.PrevSibling
	for prev != nil && prev.Type != xhtml.ElementNode {
		prev = prev.PrevSibling
	}
	if prev == nil || !slices.Contains([]atom.Atom{atom.P, atom.Span, atom.Div, atom.A}, prev.DataAtom) || len(webpage.QueryAll(prev, "time, nav, a[href]")) > 0 {
		return
	}
	label := webpage.HTMLText(prev)
	if n := len(strings.Fields(label)); n == 0 || n > 5 {
		return
	}
	art := all[i]
	if slices.ContainsFunc(jsonld.Strs(art.N, "articleSection"), func(s string) bool { return txt.NormWords(s) == txt.NormWords(label) }) {
		return
	}
	c.add(art.Path, Warning, "the page labels the article %q above its h1: set articleSection %q", label, label)
}

var (
	videoSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:www\.)?(?:youtube(?:-nocookie)?\.com/(?:embed|watch|shorts)|youtu\.be/|player\.vimeo\.com/video/|vimeo\.com/\d|fast\.wistia\.(?:net|com)/embed|player\.twitch\.tv|www\.tiktok\.com/embed)`)
	audioSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:open\.spotify\.com/embed|w\.soundcloud\.com/player|embed\.podcasts\.apple\.com|anchor\.fm/.+/embed|podcasters\.spotify\.com/.+/embed|www\.buzzsprout\.com/.+\?.*player)`)
)

// media warns on a video or audio the page plays that no VideoObject or
// AudioObject has as its contentUrl or embedUrl.
func (c *checker) media(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil {
		return
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		kind, src := "", ""
		switch n.DataAtom {
		case atom.Video, atom.Audio:
			kind = "VideoObject"
			if n.DataAtom == atom.Audio {
				kind = "AudioObject"
			}
			src, _ = webpage.Attr(n, "src")
			if s := webpage.QueryAll(n, "source[src]"); src == "" && len(s) > 0 {
				src, _ = webpage.Attr(s[0], "src")
			}
		case atom.Iframe:
			src, _ = webpage.Attr(n, "src")
			switch {
			case videoSrc.MatchString(src):
				kind = "VideoObject"
			case audioSrc.MatchString(src):
				kind = "AudioObject"
			}
		}
		if kind == "" {
			return
		}
		for _, l := range all {
			if !c.is(l.N, kind) {
				continue
			}
			if src == "" {
				return
			}
			for _, k := range []string{"contentUrl", "embedUrl", "url"} {
				if slices.ContainsFunc(jsonld.Strs(l.N, k), func(s string) bool { return p.Abs(s) == p.Abs(src) }) {
					return
				}
			}
		}
		what := src
		if what == "" {
			what = "a " + n.Data
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page plays %s but no %s has it as contentUrl or embedUrl: add one (name, description, thumbnailUrl, uploadDate) linked from the WebPage", what, kind)
	})
}

// search warns on a search box, or a link to the site's search page, with no
// SearchAction in the draft.
func (c *checker) search(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil || slices.ContainsFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "SearchAction") }) {
		return
	}
	for _, n := range webpage.QueryAll(p.Doc, `input[type="search"], [role="search"] input, input[role="searchbox"]`) {
		if webpage.Shown(n) {
			c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page has a search box but no SearchAction: add one to the WebSite's potentialAction, with a target urlTemplate holding {search_term_string}")
			return
		}
	}
	for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
		u, ok := p.SiteLink(a)
		label, _ := webpage.Attr(a, "aria-label")
		if ok && webpage.Shown(a) && slices.Contains(webpage.Segments(webpage.MustParse(u)), "search") && strings.Contains(strings.ToLower(label+" "+webpage.HTMLText(a)), "search") {
			c.add("html:"+webpage.SelectorFor(p.Doc, a), Warning, "the page links the site's search page (%s) but no SearchAction: add one to the WebSite's potentialAction, target an EntryPoint whose urlTemplate is the search page's address with {search_term_string} as the query", u)
			return
		}
	}
}

var phoneNumber = regexp.MustCompile(`\+?\d[\d ()-]{6,}\d`)

// telKey compares phone numbers by their last eight digits, so +61 4 and
// 04 forms of one number match.
func telKey(s string) string {
	d := txt.Digits(s)
	if len(d) > 8 {
		d = d[len(d)-8:]
	}
	return d
}

// contacts warns on a tel: or mailto: link no node carries, and on a
// telephone or email in the markup that the page does not show.
func (c *checker) contacts(all []jsonld.Node, p *webpage.Facts) {
	type val struct{ key, raw, path string }
	var tels, mails []val
	for _, l := range all {
		for _, s := range jsonld.Strs(l.N, "telephone") {
			if k := telKey(s); len(k) >= 6 {
				tels = append(tels, val{k, s, l.Path + ".telephone"})
			}
		}
		for _, s := range jsonld.Strs(l.N, "email") {
			mails = append(mails, val{strings.ToLower(strings.TrimPrefix(s, "mailto:")), s, l.Path + ".email"})
		}
	}
	shownTel, shownMail := map[string]bool{}, map[string]bool{}
	for _, m := range phoneNumber.FindAllString(p.Text, -1) {
		shownTel[telKey(m)] = true
	}
	if p.Doc != nil {
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			if n.DataAtom != atom.A {
				return
			}
			h, _ := webpage.Attr(n, "href")
			low := strings.ToLower(h)
			switch {
			case strings.HasPrefix(low, "tel:"):
				k := telKey(h)
				shownTel[k] = true
				if !slices.ContainsFunc(tels, func(v val) bool { return v.key == k }) {
					c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page links %s but no node has that telephone: set telephone on the business or a ContactPoint", h)
				}
			case strings.HasPrefix(low, "mailto:"):
				a, _, _ := strings.Cut(strings.TrimPrefix(low, "mailto:"), "?")
				shownMail[a] = true
				if !slices.ContainsFunc(mails, func(v val) bool { return v.key == a }) {
					c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page links %s but no node has that email: set email on the business or a ContactPoint", h)
				}
			}
		})
	}
	lowText := strings.ToLower(p.Text)
	for _, v := range tels {
		if !shownTel[v.key] {
			c.add(v.path, Warning, "telephone %s is not on the page: only a number the page shows goes in the markup", v.raw)
		}
	}
	for _, v := range mails {
		if !shownMail[v.key] && !strings.Contains(lowText, v.key) {
			c.add(v.path, Warning, "email %s is not on the page: only an address the page shows goes in the markup", v.raw)
		}
	}
}

// forms warns on a form no Action targets, and on an EntryPoint whose
// httpMethod does not match the form it targets.
// inSearch reports an element in a search landmark: a search element, or one
// with role search.
func inSearch(n *xhtml.Node) bool {
	for e := n; e != nil; e = e.Parent {
		if role, _ := webpage.Attr(e, "role"); e.Type == xhtml.ElementNode && (e.Data == "search" || role == "search") {
			return true
		}
	}
	return false
}

func (c *checker) forms(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	type target struct {
		path string
		u    *url.URL
		ep   *jsonld.Object
	}
	var targets []target
	for _, l := range all {
		if !c.is(l.N, "Action") {
			continue
		}
		raw, _ := l.N.Get("target")
		for _, v := range jsonld.Values(raw) {
			switch x := v.(type) {
			case string:
				if u, err := url.Parse(x); err == nil {
					targets = append(targets, target{l.Path + ".target", u, nil})
				}
			case *jsonld.Object:
				for _, s := range append(jsonld.Strs(x, "urlTemplate"), jsonld.Strs(x, "url")...) {
					if u, err := url.Parse(s); err == nil {
						targets = append(targets, target{d.Path[x], u, x})
					}
				}
			}
		}
	}
	warned := map[string]bool{}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.Form {
			return
		}
		// a search box is the SearchAction's; a dialog's form only closes it
		if method, _ := webpage.Attr(n, "method"); inSearch(n) || len(webpage.QueryAll(n, `input[type="search"]`)) > 0 || strings.EqualFold(strings.TrimSpace(method), "dialog") {
			return
		}
		ids := map[string]bool{}
		near := ""
		for e := n; e != nil; e = e.Parent {
			if id, ok := webpage.Attr(e, "id"); ok && id != "" && e.Type == xhtml.ElementNode {
				ids[id] = true
				if near == "" {
					near = id
				}
			}
		}
		action, _ := webpage.Attr(n, "action")
		method, _ := webpage.Attr(n, "method")
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "" {
			method = "GET"
		}
		var hit []target
		for _, t := range targets {
			if (t.u.Fragment != "" && ids[t.u.Fragment]) || (action != "" && txt.NormURL(t.u) == p.Abs(action)) {
				hit = append(hit, t)
			}
		}
		if len(hit) == 0 {
			where := "give the form or its section an id and point the target's urlTemplate at it"
			if near != "" {
				where = "with an EntryPoint target whose urlTemplate is the page URL plus #" + near
			}
			text := webpage.HTMLText(n)
			switch {
			case preOrder.MatchString(text):
				c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page's pre-order form has no Action whose target points at it: add a PreOrderAction as the Offer's potentialAction, target an EntryPoint (urlTemplate %s, httpMethod %s)", p.Abs(action), method)
			case webpage.CartForm.MatchString(action) || webpage.AddToCart.MatchString(text):
				c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page's add-to-bag form has no Action whose target points at it: add a BuyAction as the Offer's potentialAction, target an EntryPoint (urlTemplate %s, httpMethod %s, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)", p.Abs(action), method)
			case subscribeForm.MatchString(text+" "+webpage.HTMLText(n.Parent)) && len(webpage.QueryAll(n, `input[type="email"], input[name*="email"]`)) > 0:
				c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) %s", where)
			default:
				c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page's form has no Action whose target points at it: add the Action it performs (AskAction, QuoteAction, ReserveAction, SubscribeAction and the like) %s", where)
			}
			return
		}
		for _, t := range hit {
			if t.ep == nil || warned[t.path] {
				continue
			}
			hm := ""
			if v := jsonld.Strs(t.ep, "httpMethod"); len(v) > 0 {
				hm = strings.ToUpper(v[0])
			}
			switch {
			case hm == "" && method != "GET":
				warned[t.path] = true
				c.add(t.path, Warning, "the form this EntryPoint targets posts with %s: set httpMethod %s", method, method)
			case hm != "" && hm != method:
				warned[t.path] = true
				c.add(t.path+".httpMethod", Warning, "httpMethod is %s but the form it targets uses %s", hm, method)
			}
		}
	})
}

var (
	subscribeForm = regexp.MustCompile(`(?i)\b(subscribe|newsletter|sign up|join our)\b`)
)

var registration = regexp.MustCompile(`(?i)\b(ABN|ACN|ARBN|NZBN|UEN|EIN|GSTIN|VAT(?:\s+(?:no\.?|number|reg(?:istration)?\.?\s+no\.?))?|company\s+(?:no\.?|number|reg(?:istration)?\.?\s+(?:no\.?|number))|licen[cs]e\s+(?:no\.?|number))[\s:#.]*([A-Z]{0,3}\s?\d[\d ]{4,18}\d)\b`)

// registrations warns on a labelled business or licence number the page
// shows that no taxID, vatID or identifier carries.
func (c *checker) registrations(all []jsonld.Node, p *webpage.Facts) {
	var have []string
	for _, l := range all {
		for _, k := range []string{"taxID", "vatID", "identifier", "leiCode", "duns", "iso6523Code"} {
			for _, s := range jsonld.Scalars(l.N, k) {
				have = append(have, txt.Digits(s))
			}
		}
		if c.is(l.N, "PropertyValue") {
			for _, s := range jsonld.Scalars(l.N, "value") {
				have = append(have, txt.Digits(s))
			}
		}
	}
	seen := map[string]bool{}
	for _, m := range registration.FindAllStringSubmatch(p.Text, -1) {
		num := txt.Digits(m[2])
		if seen[num] || slices.Contains(have, num) {
			continue
		}
		seen[num] = true
		c.add(pagePath, Warning, "the page shows %s %s but no taxID, vatID or identifier carries it: set it on the business it identifies (taxID, or an identifier PropertyValue whose propertyID is %s)", m[1], strings.TrimSpace(m[2]), m[1])
	}
}

const otherScript = `[\p{Han}\p{Hangul}\p{Hiragana}\p{Katakana}\p{Arabic}\p{Cyrillic}\p{Greek}\p{Thai}\p{Devanagari}]`

var (
	termThenGloss = regexp.MustCompile(`(` + otherScript + `+)\s*[(（]([^()（）]{1,40})[)）]`)
	glossThenTerm = regexp.MustCompile(`\p{Latin}[\p{Latin}'’ -]{1,30}\s*[(（]\s*(` + otherScript + `+)`)
	romanised     = regexp.MustCompile(`(?i)[a-z]+[1-6]\b|[āáǎàēéěèīíǐìōóǒòūúǔùǖǘǚǜ]`)
	shortGloss    = regexp.MustCompile(`^\s*\p{Latin}[\p{Latin}'’ -]*$`)
)

// terms warns on terms the page defines that no DefinedTerm names: a <dfn>,
// a <dt> in a <dl>, and a short word in another script given with its
// romanisation or translation in brackets, either way round: 敬茶 (ging3
// caa4), red dates (紅棗). A longer run in that script is a sentence, not a
// term.
func (c *checker) terms(all []jsonld.Node, p *webpage.Facts) {
	var found []string
	add := func(t string) {
		if t = strings.TrimSpace(t); t != "" && !slices.Contains(found, t) {
			found = append(found, t)
		}
	}
	for _, m := range termThenGloss.FindAllStringSubmatch(p.Text, -1) {
		g := m[2]
		if utf8.RuneCountInString(m[1]) <= 6 && (romanised.MatchString(g) || (shortGloss.MatchString(g) && len(strings.Fields(g)) <= 4)) {
			add(m[1])
		}
	}
	for _, m := range glossThenTerm.FindAllStringSubmatch(p.Text, -1) {
		if utf8.RuneCountInString(m[1]) <= 6 {
			add(m[1])
		}
	}
	if p.Doc != nil {
		// a label that several lists repeat (Includes, How long for each
		// option) names a field of data; a glossary defines each term once
		lists := map[string]int{}
		for _, dl := range webpage.QueryAll(p.Doc, "dl") {
			seen := map[string]bool{}
			for _, d := range webpage.Pairs(dl) {
				if k := txt.NormWords(d.Label); !seen[k] {
					seen[k] = true
					lists[k]++
				}
			}
		}
		repeats := func(dl *xhtml.Node) bool {
			return slices.ContainsFunc(webpage.Pairs(dl), func(d webpage.Detail) bool { return lists[txt.NormWords(d.Label)] > 1 })
		}
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			switch n.DataAtom {
			case atom.Dfn:
				add(webpage.HTMLText(n))
			case atom.Dl:
				// a list of short values (Shipping rate: $7.95), or a
				// product's details, is data the shop checks read, and a
				// shop's dl beside a "Shop lockets" link sells: neither is a
				// glossary
				if !keyValueList(n) && p.Kind != webpage.KindProduct && !p.Promo(n) && !repeats(n) {
					for _, d := range webpage.Pairs(n) {
						add(d.Label)
					}
				}
			}
		})
	}
	var named []string
	for _, l := range all {
		if c.is(l.N, "DefinedTerm") {
			for _, k := range []string{"name", "alternateName", "termCode"} {
				named = append(named, jsonld.Strs(l.N, k)...)
			}
		}
	}
	var missing []string
	for _, t := range found {
		if !slices.ContainsFunc(named, func(s string) bool { return strings.Contains(s, t) || txt.NormWords(s) == txt.NormWords(t) }) {
			missing = append(missing, t)
		}
	}
	if len(missing) == 0 {
		return
	}
	list, more := missing, ""
	if len(list) > 12 {
		list, more = list[:12], fmt.Sprintf(" and %d more", len(missing)-12)
	}
	c.add(pagePath, Warning, "the page defines %d terms (%s%s) that no DefinedTerm names: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, alternateName for its romanisation or translation, description)", len(missing), strings.Join(list, ", "), more)
}

func attrInt(n *xhtml.Node, key string) int {
	v, _ := webpage.Attr(n, key)
	i, _ := strconv.Atoi(strings.TrimSpace(v))
	return i
}

// imgURLs are an image's src and srcset URLs, resolved.
func imgURLs(p *webpage.Facts, n *xhtml.Node) []string {
	var out []string
	if s, _ := webpage.Attr(n, "src"); s != "" {
		out = append(out, p.Abs(s))
	}
	srcset, _ := webpage.Attr(n, "srcset")
	for _, part := range strings.Split(srcset, ",") {
		if f := strings.Fields(part); len(f) > 0 {
			out = append(out, p.Abs(f[0]))
		}
	}
	return out
}

// images warns on a content image (one with alt text, outside the page's
// header, footer and navs) that no ImageObject or image URL in the draft
// names, and on an ImageObject for one that lacks its width, height or a
// description.
func (c *checker) images(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil {
		return
	}
	objs := map[string]jsonld.Node{}
	plain := map[string]bool{}
	for _, l := range all {
		if c.is(l.N, "ImageObject") {
			for _, k := range []string{"contentUrl", "url", "thumbnailUrl"} {
				for _, s := range jsonld.Strs(l.N, k) {
					objs[p.Abs(s)] = l
				}
			}
			continue
		}
		for _, k := range []string{"image", "logo", "photo", "primaryImageOfPage", "thumbnailUrl"} {
			for _, s := range jsonld.Strs(l.N, k) {
				plain[p.Abs(s)] = true
			}
		}
	}
	reported := map[string]bool{}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		// a product card's image is its Product's, which the listing check asks for
		if n.DataAtom != atom.Img || webpage.Chrome(n) || p.InCard(n) {
			return
		}
		alt, _ := webpage.Attr(n, "alt")
		w, h := attrInt(n, "width"), attrInt(n, "height")
		if strings.TrimSpace(alt) == "" || (w > 0 && w <= 2) || (h > 0 && h <= 2) {
			return
		}
		var obj *jsonld.Node
		covered := false
		for _, u := range imgURLs(p, n) {
			if o, ok := objs[u]; ok {
				obj, covered = &o, true
				break
			}
			covered = covered || plain[u]
		}
		if !covered {
			c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page's image %q has no ImageObject: add one (contentUrl, width, height, description from its alt text) as the image of the page or of the node it shows", txt.Clip(alt, 60))
			return
		}
		if obj == nil || reported[obj.Path] {
			return
		}
		var miss []string
		if w > 0 && len(jsonld.Scalars(obj.N, "width")) == 0 {
			miss = append(miss, fmt.Sprintf("width %d", w))
		}
		if h > 0 && len(jsonld.Scalars(obj.N, "height")) == 0 {
			miss = append(miss, fmt.Sprintf("height %d", h))
		}
		if len(jsonld.Strs(obj.N, "description"))+len(jsonld.Strs(obj.N, "caption")) == 0 {
			miss = append(miss, "a description from its alt text")
		}
		if len(miss) > 0 {
			reported[obj.Path] = true
			c.add(obj.Path, Warning, "this ImageObject is the page's image %q: set %s", txt.Clip(alt, 60), strings.Join(miss, ", "))
		}
	})
}

var (
	countWords = map[string]int{"two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
		"eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15, "sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19, "twenty": 20}
	countHeading = regexp.MustCompile(`(?i)\b(\d{1,3}|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)\s+(?:[\p{L}'’-]+\s+){0,2}?([\p{L}'’-]+s)\b`)
	unitNouns    = map[string]bool{"seconds": true, "secs": true, "minutes": true, "mins": true, "hours": true, "hrs": true, "days": true, "weeks": true, "months": true, "years": true,
		"guests": true, "people": true, "dollars": true, "cents": true, "percent": true, "times": true, "metres": true, "meters": true, "kilometres": true, "kilometers": true, "kms": true, "miles": true, "stars": true, "points": true,
		"pcs": true, "pieces": true, "packs": true, "pairs": true, "business": true}
)

// linkHeading reports a heading that is a link, or sits in one: a product
// card's title, not a heading over a list.
func linkHeading(n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.A {
			return true
		}
	}
	links := webpage.QueryAll(n, "a")
	return len(links) == 1 && strings.TrimSpace(webpage.HTMLText(links[0])) == strings.TrimSpace(webpage.HTMLText(n))
}

// counts warns on a heading that counts its items ("16 ideas", "Four
// steps") when no ItemList or HowTo in the draft has that many.
func (c *checker) counts(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil {
		return
	}
	have := map[int]bool{}
	for _, l := range all {
		for _, s := range jsonld.Scalars(l.N, "numberOfItems") {
			if i, err := strconv.Atoi(s); err == nil {
				have[i] = true
			}
		}
		if c.is(l.N, "ItemList") || c.is(l.N, "HowTo") {
			for _, k := range []string{"itemListElement", "step"} {
				if raw, ok := l.N.Get(k); ok {
					have[len(jsonld.Values(raw))] = true
				}
			}
		}
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4:
		default:
			return
		}
		if webpage.Chrome(n) || linkHeading(n) || p.InCard(n) {
			return
		}
		t := webpage.HTMLText(n)
		for _, m := range countHeading.FindAllStringSubmatch(t, -1) {
			if unitNouns[strings.ToLower(m[2])] {
				continue
			}
			k, err := strconv.Atoi(m[1])
			if err != nil {
				k = countWords[strings.ToLower(m[1])]
			}
			if k < 2 || have[k] || !slices.Contains(sectionCounts(n), k) {
				continue
			}
			c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the heading %q counts %d items: add an ItemList of them (numberOfItems %d, a ListItem each) or a HowTo with %d steps, in the WebPage's hasPart", txt.Clip(t, 60), k, k, k)
			return
		}
	})
}

// sectionCounts are the counts of the items a heading's section holds: its
// subheadings, and the items of the first list, the details and the table
// rows before them. A heading whose number the section does not hold counts
// something else: "2 movers", "two bedrooms", "10 business days".
func sectionCounts(h *xhtml.Node) []int {
	level := webpage.HeadingLevel(h)
	var subs, details, rows, items int
	listDone, before, started := false, true, false
	stop := false
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil && !stop; c = c.NextSibling {
			if webpage.Skipped(c) || c.Type != xhtml.ElementNode {
				continue
			}
			if c == h {
				started = true
				continue
			}
			if started {
				if l := webpage.HeadingLevel(c); l > 0 {
					if l <= level {
						stop = true
						return
					}
					if l == level+1 {
						subs++
					}
					before = false
					continue
				}
				if before {
					switch c.DataAtom {
					case atom.Ul, atom.Ol:
						if !listDone {
							items, listDone = len(webpage.QueryAll(c, "li")), true
						}
					case atom.Details:
						details++
					case atom.Tr:
						if len(webpage.QueryAll(c, "td")) > 0 {
							rows++
						}
					}
				}
			}
			walk(c)
		}
	}
	for top := h; top != nil; top = top.Parent {
		if top.Parent == nil {
			walk(top)
		}
	}
	return []int{subs, details, rows, items}
}

// outsideLinks warns on a group of three or more links to other sites, under
// one ancestor with an id, whose URLs no node in the draft names: partners,
// vendors or sources the page features.
func (c *checker) outsideLinks(p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil || p.Base == nil {
		return
	}
	type group struct{ urls, labels []string }
	var order []*xhtml.Node
	groups := map[*xhtml.Node]*group{}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.A {
			return
		}
		h, _ := webpage.Attr(n, "href")
		u, err := url.Parse(strings.TrimSpace(h))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || strings.EqualFold(u.Hostname(), p.Base.Hostname()) {
			return
		}
		// the business's profiles and share buttons have checks of their own
		if socialProfile(h) || shareLink.MatchString(h) {
			return
		}
		key := txt.NormURL(u)
		if d.URLs[key] {
			return
		}
		box := webpage.IDBox(n)
		g := groups[box]
		if g == nil {
			g = &group{}
			groups[box] = g
			order = append(order, box)
		}
		if !slices.Contains(g.urls, key) {
			g.urls = append(g.urls, key)
			label := strings.TrimSpace(webpage.HTMLText(n))
			if label == "" {
				label, _ = webpage.Attr(n, "aria-label")
			}
			if label = txt.Clip(label, 40); label != "" && !slices.Contains(g.labels, label) {
				g.labels = append(g.labels, label)
			}
		}
	})
	for _, box := range order {
		g := groups[box]
		if len(g.urls) < 3 {
			continue
		}
		labels := g.labels
		if len(labels) > 8 {
			labels = append(labels[:8:8], "…")
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "%d links to outside sites here (%s) name no node: when they are businesses or people the page features, add an ItemList of them to the WebPage's hasPart, each with url and sameAs; when they are sources, list them as citation", len(g.urls), strings.Join(labels, ", "))
	}
}

// related warns when a block the page labels as related reading links to
// pages of the site that the WebPage's relatedLink does not list. A block of
// product cards is the Product's related products instead.
func (c *checker) related(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	var rel []string
	for _, l := range all {
		if c.is(l.N, "WebPage") {
			for _, s := range jsonld.Strs(l.N, "relatedLink") {
				rel = append(rel, p.Abs(s))
			}
		}
	}
	for _, b := range p.RelatedBoxes() {
		t, box := b.Label, b.N
		var cards []*webpage.Card
		for _, cd := range p.ProductCards() {
			if cd.Related && slices.Contains(webpage.Ancestors(cd.N), box) {
				cards = append(cards, cd)
			}
		}
		if len(cards) > 0 {
			var names []string
			for _, cd := range cards {
				if !d.URLs[cd.URL] && !slices.Contains(names, cd.Name) {
					names = append(names, cd.Name)
				}
			}
			if len(names) > 0 {
				c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "the page's %q block lists products (%s) that the Product's isRelatedTo and isSimilarTo do not name: add each as a Product (name, url, image, offers)", t, strings.Join(names, ", "))
			}
			continue
		}
		// a guide's own pages, listed on its hub, are a part of the page
		if webpage.GuideLabel.MatchString(t) {
			var pages []string
			for _, u := range p.SiteLinks(box) {
				if !slices.Contains(pages, u) {
					pages = append(pages, u)
				}
			}
			if len(pages) >= 3 && slices.ContainsFunc(pages, func(u string) bool { return !d.URLs[u] }) {
				c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "the page's %q block lists %d pages of the site: add an ItemList of them (a ListItem for each with position, url and name) to the WebPage's hasPart; a hub page whose purpose is the list is a CollectionPage with the ItemList as its mainEntity", t, len(pages))
			}
			continue
		}
		var missing []string
		webpage.Visit(box, func(a *xhtml.Node) {
			if a.DataAtom != atom.A {
				return
			}
			h, _ := webpage.Attr(a, "href")
			pu, err := url.Parse(strings.TrimSpace(h))
			if h == "" || strings.HasPrefix(h, "#") || err != nil {
				return
			}
			if p.Base != nil {
				pu = p.Base.ResolveReference(pu)
			}
			if (p.Base != nil && !strings.EqualFold(pu.Hostname(), p.Base.Hostname())) || (p.Base == nil && pu.IsAbs()) {
				return
			}
			if link := pu.String(); !slices.Contains(rel, txt.NormURL(pu)) && !slices.Contains(missing, link) {
				missing = append(missing, link)
			}
		})
		if len(missing) > 0 {
			c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "the page's %q links are not in the WebPage's relatedLink: add %s", t, strings.Join(missing, ", "))
		}
	}
}

var responseTime = regexp.MustCompile(`(?i)\b(?:repl(?:y|ies)|responds?|responses?|get back to you|hear back)\b[^.!?]{0,40}?\bwithin\s+(?:(?:\d+|one|two|three)\s+)?(?:business days?|working days?|days?|hours?|minutes?)`)

// responseTime warns when the page promises a reply time and no
// ServiceChannel has it as processingTime.
func (c *checker) responseTime(all []jsonld.Node, p *webpage.Facts) {
	m := responseTime.FindString(p.Text)
	if m == "" || slices.ContainsFunc(all, func(l jsonld.Node) bool { return len(jsonld.Scalars(l.N, "processingTime")) > 0 }) {
		return
	}
	path := pagePath
	if i := slices.IndexFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "ServiceChannel") }); i >= 0 {
		path = all[i].Path
	}
	c.add(path, Warning, "the page promises %q but no ServiceChannel has processingTime: set it as a Duration (P1D for a day) on the channel the promise is for", m)
}

// attribution is who a blockquote is from: a cite inside it, or after it in
// its figure.
func attribution(n *xhtml.Node) string {
	for e := n; e != nil && e.DataAtom != atom.Body; e = e.Parent {
		if cs := webpage.QueryAll(e, "cite"); len(cs) > 0 {
			return " (" + strings.TrimSpace(webpage.HTMLText(cs[0])) + ")"
		}
		if e.DataAtom == atom.Figure {
			break
		}
	}
	return ""
}

// quotes warns on a quotation the page shows that no string in the draft
// carries: a blockquote, or a paragraph that quotes someone it names. A
// testimonial is a Review, another quote a Quotation. A string of three
// words or more carries a quote it contains; a shorter one only the quote it
// is.
func (c *checker) quotes(p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	var carried []string
	exact := map[string]bool{}
	for _, s := range d.Strs {
		w := txt.NormWords(txt.StripTags(s))
		exact[w] = true
		if strings.Count(w, " ") >= 2 {
			carried = append(carried, " "+w+" ")
		}
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		text, who := webpage.HTMLText(n), ""
		switch {
		case n.DataAtom == atom.Blockquote:
			who = attribution(n)
		case n.DataAtom == atom.P:
			var ok bool
			if text, who, ok = pullQuote(n); !ok {
				return
			}
			who = " (" + who + ")"
		default:
			return
		}
		q := txt.NormWords(text)
		if q == "" || exact[q] || slices.ContainsFunc(carried, func(s string) bool { return strings.Contains(s, " "+q+" ") }) {
			return
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page quotes %q%s and no node carries it: a customer's opinion of a business, product or service is a Review of it (reviewBody, author, itemReviewed); another quotation is a Quotation (text, creator)", txt.Clip(text, 60), who)
	})
}

// quoteMarks pairs each opening quotation mark with its closing one.
var quoteMarks = map[string]string{"“": "”", `"`: `"`, "«": "»", "「": "」"}

// pullQuote is a paragraph, outside a blockquote, that quotes someone: its
// text opens with a quotation mark, and a <cite> in it or beside it, or the
// text after the closing mark when it starts with @ or a dash, says who. It
// is the words between the marks and who said them.
func pullQuote(n *xhtml.Node) (quote, who string, ok bool) {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.Blockquote {
			return "", "", false
		}
	}
	t := strings.TrimSpace(webpage.HTMLText(n))
	for open, shut := range quoteMarks {
		if !strings.HasPrefix(t, open) {
			continue
		}
		end := strings.Index(t[len(open):], shut)
		if end < 0 {
			return "", "", false
		}
		quote = strings.TrimSpace(t[len(open) : len(open)+end])
		rest := strings.TrimSpace(t[len(open)+end+len(shut):])
		cites := webpage.QueryAll(n, "cite")
		if n.Parent != nil {
			for ch := n.Parent.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.DataAtom == atom.Cite {
					cites = append(cites, ch)
				}
			}
		}
		switch {
		case len(cites) > 0:
			who = strings.TrimSpace(webpage.HTMLText(cites[0]))
		case strings.HasPrefix(rest, "@") || strings.HasPrefix(rest, "—") || strings.HasPrefix(rest, "–") || strings.HasPrefix(rest, "-"):
			who = strings.TrimSpace(strings.TrimLeft(rest, "—–- "))
		}
		return quote, who, quote != "" && who != ""
	}
	return "", "", false
}

var (
	integerOutOf = regexp.MustCompile(`^\d+\s*/`)
	ratingWord   = regexp.MustCompile(`(?i)\b(rat(?:ed|ing|ings)|scores?|stars?|reviews?)\b`)
)

var ratingShown = regexp.MustCompile(`(?i)\b(\d(?:\.\d)?)\s*(?:/\s*(?:5|10)\b|out of\s*(?:5|10)\b|stars?\b)`)

// ratings warns on ratings the page shows that no Rating carries as its
// ratingValue.
func (c *checker) ratings(all []jsonld.Node, p *webpage.Facts) {
	var shownVals [][]string
	for _, loc := range ratingShown.FindAllStringSubmatchIndex(p.Text, -1) {
		m := p.Text[loc[0]:loc[1]]
		// "1 / 10" with no rating word before it is a gallery's counter
		if integerOutOf.MatchString(m) && !ratingWord.MatchString(p.Text[max(0, loc[0]-30):loc[0]]) {
			continue
		}
		shownVals = append(shownVals, []string{m, p.Text[loc[2]:loc[3]]})
	}
	if p.Doc != nil {
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			for _, k := range []string{"aria-label", "title"} {
				if v, ok := webpage.Attr(n, k); ok {
					shownVals = append(shownVals, ratingShown.FindAllStringSubmatch(v, -1)...)
				}
			}
		})
	}
	have := map[float64]bool{}
	for _, l := range all {
		if c.is(l.N, "Rating") {
			for _, s := range jsonld.Scalars(l.N, "ratingValue") {
				if f, ok := txt.Amount(s); ok {
					have[f] = true
				}
			}
		}
	}
	var missing []string
	for _, m := range shownVals {
		if f, ok := txt.Amount(m[1]); ok && !have[f] && !slices.Contains(missing, m[1]) {
			missing = append(missing, m[1])
		}
	}
	if len(missing) > 0 {
		c.add(pagePath, Warning, "the page shows ratings (%s) that no Rating or AggregateRating carries as ratingValue: mark each up on the Review or item it rates (ratingValue, bestRating, and ratingCount for an aggregate)", strings.Join(missing, ", "))
	}
}
