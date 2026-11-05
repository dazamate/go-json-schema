package vocabtool

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/andybalholm/cascadia"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The content checks hold the markup to the rendered page in both
// directions. Text the markup repeats must be on the page (a HowToStep, a
// Question, an Answer, a Review), and what the page shows that a type or a
// property describes must be in the markup (a copyright line, a reading time,
// images, a video, prices, ratings, quotations, terms, counted lists, links,
// a search box, a form, a phone number, a registration number).

const pagePath = "html:body" // an issue about the page as a whole

// pageFacts is what the content checks read from the rendered page.
type pageFacts struct {
	doc     *xhtml.Node // nil when only the page's text is known
	text    string      // the visible text, whitespace collapsed
	norm    string      // normWords(text), padded with spaces
	main    string      // the visible text outside the site's header, footer and navs
	base    *url.URL    // the page's canonical URL, else its address, or nil
	url     *url.URL    // the page's address, else its canonical URL, or nil
	kind    pageKind
	private string           // why search engines skip the page ("" when they index it)
	claimed map[float64]bool // prices a check other than prices reports

	cards     []*card // product cards, found once
	cardsDone bool
}

func newPageFacts(doc *xhtml.Node, text, address string) *pageFacts {
	p := &pageFacts{doc: doc, text: text, norm: " " + normWords(text) + " ", main: text, claimed: map[float64]bool{}}
	if u, err := url.Parse(strings.TrimSpace(address)); err == nil && u.IsAbs() {
		p.url = u
	}
	if doc != nil {
		p.base = canonical(doc)
		p.main = mainText(doc)
	}
	if p.base == nil {
		p.base = p.url
	}
	if p.url == nil {
		p.url = p.base
	}
	p.kind = p.classify()
	switch {
	case doc != nil && noindex(doc):
		p.private = "noindex"
	case privateKinds[p.kind] != "":
		p.private = privateKinds[p.kind]
	}
	return p
}

// mainText is the visible text outside the site's header, footer and navs.
func mainText(doc *xhtml.Node) string { return strings.Join(mainLines(doc), " ") }

// lineBreaks are the elements that start and end a line of text.
var lineBreaks = map[atom.Atom]bool{
	atom.Address: true, atom.Article: true, atom.Aside: true, atom.Blockquote: true, atom.Br: true, atom.Caption: true,
	atom.Dd: true, atom.Details: true, atom.Div: true, atom.Dl: true, atom.Dt: true, atom.Fieldset: true,
	atom.Figcaption: true, atom.Figure: true, atom.Footer: true, atom.Form: true, atom.H1: true, atom.H2: true,
	atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true, atom.Header: true, atom.Hr: true, atom.Legend: true,
	atom.Li: true, atom.Main: true, atom.Ol: true, atom.P: true, atom.Pre: true, atom.Section: true, atom.Summary: true,
	atom.Table: true, atom.Td: true, atom.Th: true, atom.Tr: true, atom.Ul: true,
}

// mainLines is mainText a line at a time: a block element starts and ends a
// line, so a line never runs into the next block's words.
func mainLines(doc *xhtml.Node) []string {
	var lines []string
	var b strings.Builder
	end := func() {
		if l := strings.Join(strings.Fields(b.String()), " "); l != "" {
			lines = append(lines, l)
		}
		b.Reset()
	}
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if skipped(ch) || ch.DataAtom == atom.Nav || ((ch.DataAtom == atom.Header || ch.DataAtom == atom.Footer) && !scoped(ch)) {
				continue
			}
			if ch.Type == xhtml.TextNode {
				b.WriteString(ch.Data)
				b.WriteByte(' ')
			}
			block := ch.Type == xhtml.ElementNode && lineBreaks[ch.DataAtom]
			if block {
				end()
			}
			walk(ch)
			if block {
				end()
			}
		}
	}
	walk(doc)
	end()
	return lines
}

// canonical is the page's canonical link, or its og:url.
func canonical(doc *xhtml.Node) *url.URL {
	for _, sel := range []string{`link[rel~="canonical"]`, `meta[property="og:url"]`} {
		for _, n := range queryAll(doc, sel) {
			v, _ := attr(n, "href")
			if v == "" {
				v, _ = attr(n, "content")
			}
			if u, err := url.Parse(strings.TrimSpace(v)); err == nil && u.IsAbs() {
				return u
			}
		}
	}
	return nil
}

func queryAll(n *xhtml.Node, sel string) []*xhtml.Node {
	return cascadia.QueryAll(n, cascadia.MustCompile(sel))
}

// shown reports whether a visitor can see the element: no ancestor of it is
// skipped.
func shown(n *xhtml.Node) bool {
	for e := n; e != nil; e = e.Parent {
		if skipped(e) {
			return false
		}
	}
	return true
}

// abs resolves a link on the page to a normalized absolute URL.
func (p *pageFacts) abs(ref string) string {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ref
	}
	if p.base != nil {
		u = p.base.ResolveReference(u)
	}
	return normURL(u)
}

// normURL drops the fragment, lowercases the scheme and host, and drops a
// trailing slash, so links to one page compare equal.
func normURL(u *url.URL) string {
	v := *u
	v.Fragment, v.RawFragment = "", ""
	v.Scheme, v.Host = strings.ToLower(v.Scheme), strings.ToLower(v.Host)
	if len(v.Path) > 1 {
		v.Path = strings.TrimSuffix(v.Path, "/")
		v.RawPath = ""
	}
	if v.Path == "/" {
		v.Path = ""
	}
	return v.String()
}

func normLink(s string) string {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return s
	}
	return normURL(u)
}

// draftIndex holds what the content checks look up in the draft.
type draftIndex struct {
	strs []string           // every string value
	urls map[string]bool    // every string value, as a normalized URL
	path map[*object]string // every node's path
}

func indexDraft(all []located) *draftIndex {
	d := &draftIndex{urls: map[string]bool{}, path: map[*object]string{}}
	for _, l := range all {
		d.path[l.n] = l.path
		for _, k := range l.n.keys {
			if strings.HasPrefix(k, "@") {
				continue
			}
			for _, v := range values(l.n.vals[k]) {
				if s, ok := v.(string); ok {
					d.strs = append(d.strs, s)
					d.urls[normLink(s)] = true
				}
			}
		}
	}
	return d
}

// is reports whether the node has the type or a subtype of it.
func (c *checker) is(n *object, ty string) bool {
	return c.v.Is(typeNames(n), ty)
}

// scalars are a property's string, number and boolean values, as text.
func scalars(n *object, key string) []string {
	raw, _ := n.get(key)
	var out []string
	for _, v := range values(raw) {
		switch x := v.(type) {
		case string:
			out = append(out, x)
		case fmt.Stringer: // json.Number
			out = append(out, x.String())
		case bool:
			out = append(out, strconv.FormatBool(x))
		}
	}
	return out
}

var tags = regexp.MustCompile(`<[^>]*>`)

// stripTags is the text of a value that may hold HTML.
func stripTags(s string) string { return html.UnescapeString(tags.ReplaceAllString(s, " ")) }

// clip shortens s to n runes.
func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// pageNode is the path of the WebPage the draft describes the page with:
// the one whose url is the canonical URL, or else the first WebPage.
func (c *checker) pageNode(all []located, p *pageFacts) string {
	first := ""
	for _, l := range all {
		if !c.is(l.n, "WebPage") || len(l.n.keys) < 2 {
			continue
		}
		if p.base != nil && slices.ContainsFunc(strs(l.n, "url"), func(s string) bool { return normLink(s) == normURL(p.base) }) {
			return l.path
		}
		if first == "" {
			first = l.path
		}
	}
	if first == "" {
		return "$"
	}
	return first
}

// privateNotes say what a page search engines skip would be typed as.
var privateNotes = map[pageKind]string{
	kindSearch:   "its WebPage is a SearchResultsPage",
	kindCheckout: "its WebPage is a CheckoutPage",
}

// content runs the checks that compare the draft with the page. What the
// markup repeats is held to the page always; what the page shows is asked of
// the markup only on a page search engines index.
func (c *checker) content(all []located, p *pageFacts) {
	d := indexDraft(all)
	c.shownText(all, p)
	c.anchors(all, p)
	c.offers(all)
	c.personNames(all)
	c.pageNames(all)
	if p.private != "" {
		what := p.private
		if p.private == "noindex" {
			what = "the page is noindex"
		} else {
			what = "the page is " + what
		}
		note := ""
		if n := privateNotes[p.kind]; n != "" {
			note = "; if you mark it up at all, " + n
		}
		c.add(pagePath, Info, "%s: search engines leave its structured data out, so the shop checks skip it%s", what, note)
		return
	}
	c.shop(all, p, d)
	c.site(all, p, d)
	c.sources(all, p, d)
	c.data(all, p, d)
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
	c.costs(all, p)
	c.prices(all, p)
	c.quotes(p, d)
	c.ratings(all, p)
}

// shop runs the shop checks: the sitewide nodes, the page's type, and what a
// product page, a listing and the shop's policies show. They run before
// prices, which leaves out the prices they report.
func (c *checker) shop(all []located, p *pageFacts, d *draftIndex) {
	if p.doc != nil {
		c.pageType(all, p)
		c.website(all, p)
		if p.sells() || p.address() != "" {
			c.business(all, p)
		}
		c.logo(all, p)
		c.social(p, d)
		c.catalog(all, p)
	}
	if p.kind == kindProduct {
		c.product(all, p, d)
	}
	c.listing(all, p, d)
	// what a shop's pages state about its own sales; on another site the
	// same words ("a damaged item", "pay by bank transfer") are about others
	if p.sells() {
		c.sale(all, p)
		c.stock(all, p)
		c.members(all, p)
		c.extras(all, p)
		c.shipping(all, p)
		c.returns(all, p)
		c.payments(all, p)
	}
	c.engagement(all, p)
	c.whereAndWhen(all, p)
	c.faq(all, p)
	c.founding(all, p)
}

// shownProps are the texts the markup must repeat word for word.
var shownProps = []struct{ typ, prop string }{
	{"HowToStep", "name"}, {"HowToStep", "text"}, {"HowToSection", "name"},
	{"HowToDirection", "text"}, {"HowToTip", "text"},
	{"Question", "name"}, {"Answer", "text"},
	{"Review", "reviewBody"}, {"Quotation", "text"},
}

// shownText warns on text the markup gives as the page's that the page does
// not show: a HowTo step, a question, an answer, a review.
func (c *checker) shownText(all []located, p *pageFacts) {
	for _, l := range all {
		for _, sp := range shownProps {
			if !c.is(l.n, sp.typ) {
				continue
			}
			for _, s := range strs(l.n, sp.prop) {
				if w := normWords(stripTags(s)); w != "" && !strings.Contains(p.norm, " "+w+" ") {
					c.add(l.path+"."+sp.prop, Warning, "%s %s %q is not on the page: the markup must repeat the visible text", sp.typ, sp.prop, clip(s, 80))
				}
			}
		}
	}
}

// anchors warns on a link to this page whose #fragment is no id on it.
func (c *checker) anchors(all []located, p *pageFacts) {
	if p.doc == nil || p.base == nil {
		return
	}
	ids := map[string]bool{}
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if id, ok := attr(n, "id"); ok && n.Type == xhtml.ElementNode {
			ids[id] = true
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(p.doc)
	page := normURL(p.base)
	for _, l := range all {
		for _, k := range l.n.keys {
			if strings.HasPrefix(k, "@") || k == "sameAs" {
				continue
			}
			for _, v := range values(l.n.vals[k]) {
				s, _ := v.(string)
				u, err := url.Parse(s)
				if s == "" || err != nil || u.Fragment == "" || !u.IsAbs() || normURL(u) != page {
					continue
				}
				if !ids[u.Fragment] {
					c.add(l.path+"."+k, Warning, "%s points at #%s, which is no id on the page", s, u.Fragment)
				}
			}
		}
	}
}

var copyrightLine = regexp.MustCompile(`(?i)(?:©|\(c\)|copyright)\s*(?:\d{4}\s*[-–]\s*)?(\d{4})`)

// copyright warns when the page shows a copyright year the markup lacks.
func (c *checker) copyright(all []located, p *pageFacts) {
	m := copyrightLine.FindStringSubmatch(p.text)
	if m == nil {
		return
	}
	hasYear, hasHolder := false, false
	for _, l := range all {
		hasYear = hasYear || slices.Contains(scalars(l.n, "copyrightYear"), m[1])
		if _, ok := l.n.get("copyrightHolder"); ok {
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
func (c *checker) readTime(all []located, p *pageFacts) {
	m := readingTime.FindStringSubmatch(p.text)
	if m == nil {
		return
	}
	want := "PT" + m[1] + "M"
	target := ""
	for _, l := range all {
		if !c.is(l.n, "CreativeWork") {
			continue
		}
		for _, v := range scalars(l.n, "timeRequired") {
			if !strings.EqualFold(v, want) {
				c.add(l.path+".timeRequired", Warning, "timeRequired is %s but the page says %q: make it %s", v, m[0], want)
			}
			return
		}
		if target == "" && c.is(l.n, "Article") {
			target = l.path
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
func (c *checker) sectionLabel(all []located, p *pageFacts) {
	if p.doc == nil {
		return
	}
	i := slices.IndexFunc(all, func(l located) bool { return c.is(l.n, "Article") })
	if i < 0 {
		return
	}
	var h1 *xhtml.Node
	visit(p.doc, func(n *xhtml.Node) {
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
	if prev == nil || !slices.Contains([]atom.Atom{atom.P, atom.Span, atom.Div, atom.A}, prev.DataAtom) || len(queryAll(prev, "time, nav, a[href]")) > 0 {
		return
	}
	label := htmlText(prev)
	if n := len(strings.Fields(label)); n == 0 || n > 5 {
		return
	}
	art := all[i]
	if slices.ContainsFunc(strs(art.n, "articleSection"), func(s string) bool { return normWords(s) == normWords(label) }) {
		return
	}
	c.add(art.path, Warning, "the page labels the article %q above its h1: set articleSection %q", label, label)
}

var (
	videoSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:www\.)?(?:youtube(?:-nocookie)?\.com/(?:embed|watch|shorts)|youtu\.be/|player\.vimeo\.com/video/|vimeo\.com/\d|fast\.wistia\.(?:net|com)/embed|player\.twitch\.tv|www\.tiktok\.com/embed)`)
	audioSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:open\.spotify\.com/embed|w\.soundcloud\.com/player|embed\.podcasts\.apple\.com|anchor\.fm/.+/embed|podcasters\.spotify\.com/.+/embed|www\.buzzsprout\.com/.+\?.*player)`)
)

// media warns on a video or audio the page plays that no VideoObject or
// AudioObject has as its contentUrl or embedUrl.
func (c *checker) media(all []located, p *pageFacts) {
	if p.doc == nil {
		return
	}
	visit(p.doc, func(n *xhtml.Node) {
		kind, src := "", ""
		switch n.DataAtom {
		case atom.Video, atom.Audio:
			kind = "VideoObject"
			if n.DataAtom == atom.Audio {
				kind = "AudioObject"
			}
			src, _ = attr(n, "src")
			if s := queryAll(n, "source[src]"); src == "" && len(s) > 0 {
				src, _ = attr(s[0], "src")
			}
		case atom.Iframe:
			src, _ = attr(n, "src")
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
			if !c.is(l.n, kind) {
				continue
			}
			if src == "" {
				return
			}
			for _, k := range []string{"contentUrl", "embedUrl", "url"} {
				if slices.ContainsFunc(strs(l.n, k), func(s string) bool { return p.abs(s) == p.abs(src) }) {
					return
				}
			}
		}
		what := src
		if what == "" {
			what = "a " + n.Data
		}
		c.add("html:"+selectorFor(p.doc, n), Warning, "the page plays %s but no %s has it as contentUrl or embedUrl: add one (name, description, thumbnailUrl, uploadDate) linked from the WebPage", what, kind)
	})
}

// search warns on a search box, or a link to the site's search page, with no
// SearchAction in the draft.
func (c *checker) search(all []located, p *pageFacts) {
	if p.doc == nil || slices.ContainsFunc(all, func(l located) bool { return c.is(l.n, "SearchAction") }) {
		return
	}
	for _, n := range queryAll(p.doc, `input[type="search"], [role="search"] input, input[role="searchbox"]`) {
		if shown(n) {
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page has a search box but no SearchAction: add one to the WebSite's potentialAction, with a target urlTemplate holding {search_term_string}")
			return
		}
	}
	for _, a := range queryAll(p.doc, "a[href]") {
		u, ok := p.siteLink(a)
		label, _ := attr(a, "aria-label")
		if ok && shown(a) && slices.Contains(segments(mustParse(u)), "search") && strings.Contains(strings.ToLower(label+" "+htmlText(a)), "search") {
			c.add("html:"+selectorFor(p.doc, a), Warning, "the page links the site's search page (%s) but no SearchAction: add one to the WebSite's potentialAction, target an EntryPoint whose urlTemplate is the search page's address with {search_term_string} as the query", u)
			return
		}
	}
}

// mustParse parses a URL this package normalized.
func mustParse(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

var phoneNumber = regexp.MustCompile(`\+?\d[\d ()-]{6,}\d`)

// digits are the digits in s.
func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// telKey compares phone numbers by their last eight digits, so +61 4 and
// 04 forms of one number match.
func telKey(s string) string {
	d := digits(s)
	if len(d) > 8 {
		d = d[len(d)-8:]
	}
	return d
}

// contacts warns on a tel: or mailto: link no node carries, and on a
// telephone or email in the markup that the page does not show.
func (c *checker) contacts(all []located, p *pageFacts) {
	type val struct{ key, raw, path string }
	var tels, mails []val
	for _, l := range all {
		for _, s := range strs(l.n, "telephone") {
			if k := telKey(s); len(k) >= 6 {
				tels = append(tels, val{k, s, l.path + ".telephone"})
			}
		}
		for _, s := range strs(l.n, "email") {
			mails = append(mails, val{strings.ToLower(strings.TrimPrefix(s, "mailto:")), s, l.path + ".email"})
		}
	}
	shownTel, shownMail := map[string]bool{}, map[string]bool{}
	for _, m := range phoneNumber.FindAllString(p.text, -1) {
		shownTel[telKey(m)] = true
	}
	if p.doc != nil {
		visit(p.doc, func(n *xhtml.Node) {
			if n.DataAtom != atom.A {
				return
			}
			h, _ := attr(n, "href")
			low := strings.ToLower(h)
			switch {
			case strings.HasPrefix(low, "tel:"):
				k := telKey(h)
				shownTel[k] = true
				if !slices.ContainsFunc(tels, func(v val) bool { return v.key == k }) {
					c.add("html:"+selectorFor(p.doc, n), Warning, "the page links %s but no node has that telephone: set telephone on the business or a ContactPoint", h)
				}
			case strings.HasPrefix(low, "mailto:"):
				a, _, _ := strings.Cut(strings.TrimPrefix(low, "mailto:"), "?")
				shownMail[a] = true
				if !slices.ContainsFunc(mails, func(v val) bool { return v.key == a }) {
					c.add("html:"+selectorFor(p.doc, n), Warning, "the page links %s but no node has that email: set email on the business or a ContactPoint", h)
				}
			}
		})
	}
	lowText := strings.ToLower(p.text)
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
		if role, _ := attr(e, "role"); e.Type == xhtml.ElementNode && (e.Data == "search" || role == "search") {
			return true
		}
	}
	return false
}

func (c *checker) forms(all []located, p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	type target struct {
		path string
		u    *url.URL
		ep   *object
	}
	var targets []target
	for _, l := range all {
		if !c.is(l.n, "Action") {
			continue
		}
		raw, _ := l.n.get("target")
		for _, v := range values(raw) {
			switch x := v.(type) {
			case string:
				if u, err := url.Parse(x); err == nil {
					targets = append(targets, target{l.path + ".target", u, nil})
				}
			case *object:
				for _, s := range append(strs(x, "urlTemplate"), strs(x, "url")...) {
					if u, err := url.Parse(s); err == nil {
						targets = append(targets, target{d.path[x], u, x})
					}
				}
			}
		}
	}
	warned := map[string]bool{}
	visit(p.doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.Form {
			return
		}
		// a search box is the SearchAction's; a dialog's form only closes it
		if method, _ := attr(n, "method"); inSearch(n) || len(queryAll(n, `input[type="search"]`)) > 0 || strings.EqualFold(strings.TrimSpace(method), "dialog") {
			return
		}
		ids := map[string]bool{}
		near := ""
		for e := n; e != nil; e = e.Parent {
			if id, ok := attr(e, "id"); ok && id != "" && e.Type == xhtml.ElementNode {
				ids[id] = true
				if near == "" {
					near = id
				}
			}
		}
		action, _ := attr(n, "action")
		method, _ := attr(n, "method")
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "" {
			method = "GET"
		}
		var hit []target
		for _, t := range targets {
			if (t.u.Fragment != "" && ids[t.u.Fragment]) || (action != "" && normURL(t.u) == p.abs(action)) {
				hit = append(hit, t)
			}
		}
		if len(hit) == 0 {
			where := "give the form or its section an id and point the target's urlTemplate at it"
			if near != "" {
				where = "with an EntryPoint target whose urlTemplate is the page URL plus #" + near
			}
			text := htmlText(n)
			switch {
			case preOrder.MatchString(text):
				c.add("html:"+selectorFor(p.doc, n), Warning, "the page's pre-order form has no Action whose target points at it: add a PreOrderAction as the Offer's potentialAction, target an EntryPoint (urlTemplate %s, httpMethod %s)", p.abs(action), method)
			case cartForm.MatchString(action) || addToCart.MatchString(text):
				c.add("html:"+selectorFor(p.doc, n), Warning, "the page's add-to-bag form has no Action whose target points at it: add a BuyAction as the Offer's potentialAction, target an EntryPoint (urlTemplate %s, httpMethod %s, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)", p.abs(action), method)
			case subscribeForm.MatchString(text+" "+htmlText(n.Parent)) && len(queryAll(n, `input[type="email"], input[name*="email"]`)) > 0:
				c.add("html:"+selectorFor(p.doc, n), Warning, "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) %s", where)
			default:
				c.add("html:"+selectorFor(p.doc, n), Warning, "the page's form has no Action whose target points at it: add the Action it performs (AskAction, QuoteAction, ReserveAction, SubscribeAction and the like) %s", where)
			}
			return
		}
		for _, t := range hit {
			if t.ep == nil || warned[t.path] {
				continue
			}
			hm := ""
			if v := strs(t.ep, "httpMethod"); len(v) > 0 {
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
	cartForm      = regexp.MustCompile(`(?i)(cart|basket|bag)\b`)
	subscribeForm = regexp.MustCompile(`(?i)\b(subscribe|newsletter|sign up|join our)\b`)
)

var registration = regexp.MustCompile(`(?i)\b(ABN|ACN|ARBN|NZBN|UEN|EIN|GSTIN|VAT(?:\s+(?:no\.?|number|reg(?:istration)?\.?\s+no\.?))?|company\s+(?:no\.?|number|reg(?:istration)?\.?\s+(?:no\.?|number))|licen[cs]e\s+(?:no\.?|number))[\s:#.]*([A-Z]{0,3}\s?\d[\d ]{4,18}\d)\b`)

// registrations warns on a labelled business or licence number the page
// shows that no taxID, vatID or identifier carries.
func (c *checker) registrations(all []located, p *pageFacts) {
	var have []string
	for _, l := range all {
		for _, k := range []string{"taxID", "vatID", "identifier", "leiCode", "duns", "iso6523Code"} {
			for _, s := range scalars(l.n, k) {
				have = append(have, digits(s))
			}
		}
		if c.is(l.n, "PropertyValue") {
			for _, s := range scalars(l.n, "value") {
				have = append(have, digits(s))
			}
		}
	}
	seen := map[string]bool{}
	for _, m := range registration.FindAllStringSubmatch(p.text, -1) {
		num := digits(m[2])
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
func (c *checker) terms(all []located, p *pageFacts) {
	var found []string
	add := func(t string) {
		if t = strings.TrimSpace(t); t != "" && !slices.Contains(found, t) {
			found = append(found, t)
		}
	}
	for _, m := range termThenGloss.FindAllStringSubmatch(p.text, -1) {
		g := m[2]
		if utf8.RuneCountInString(m[1]) <= 6 && (romanised.MatchString(g) || (shortGloss.MatchString(g) && len(strings.Fields(g)) <= 4)) {
			add(m[1])
		}
	}
	for _, m := range glossThenTerm.FindAllStringSubmatch(p.text, -1) {
		if utf8.RuneCountInString(m[1]) <= 6 {
			add(m[1])
		}
	}
	if p.doc != nil {
		// a label that several lists repeat (Includes, How long for each
		// option) names a field of data; a glossary defines each term once
		lists := map[string]int{}
		for _, dl := range queryAll(p.doc, "dl") {
			seen := map[string]bool{}
			for _, d := range pairs(dl) {
				if k := normWords(d.label); !seen[k] {
					seen[k] = true
					lists[k]++
				}
			}
		}
		repeats := func(dl *xhtml.Node) bool {
			return slices.ContainsFunc(pairs(dl), func(d detail) bool { return lists[normWords(d.label)] > 1 })
		}
		visit(p.doc, func(n *xhtml.Node) {
			switch n.DataAtom {
			case atom.Dfn:
				add(htmlText(n))
			case atom.Dl:
				// a list of short values (Shipping rate: $7.95), or a
				// product's details, is data the shop checks read, and a
				// shop's dl beside a "Shop lockets" link sells: neither is a
				// glossary
				if !keyValueList(n) && p.kind != kindProduct && !p.promo(n) && !repeats(n) {
					for _, d := range pairs(n) {
						add(d.label)
					}
				}
			}
		})
	}
	var named []string
	for _, l := range all {
		if c.is(l.n, "DefinedTerm") {
			for _, k := range []string{"name", "alternateName", "termCode"} {
				named = append(named, strs(l.n, k)...)
			}
		}
	}
	var missing []string
	for _, t := range found {
		if !slices.ContainsFunc(named, func(s string) bool { return strings.Contains(s, t) || normWords(s) == normWords(t) }) {
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

// chrome reports an element in the page's own header, footer or a nav.
func chrome(n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		switch e.DataAtom {
		case atom.Nav:
			return true
		case atom.Header, atom.Footer:
			if !scoped(e) {
				return true
			}
		}
	}
	return false
}

// siteChromeAround reports an element in the site's own header or footer.
func siteChromeAround(n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if (e.DataAtom == atom.Header || e.DataAtom == atom.Footer) && !scoped(e) {
			return true
		}
	}
	return false
}

func attrInt(n *xhtml.Node, key string) int {
	v, _ := attr(n, key)
	i, _ := strconv.Atoi(strings.TrimSpace(v))
	return i
}

// imgURLs are an image's src and srcset URLs, resolved.
func (p *pageFacts) imgURLs(n *xhtml.Node) []string {
	var out []string
	if s, _ := attr(n, "src"); s != "" {
		out = append(out, p.abs(s))
	}
	srcset, _ := attr(n, "srcset")
	for _, part := range strings.Split(srcset, ",") {
		if f := strings.Fields(part); len(f) > 0 {
			out = append(out, p.abs(f[0]))
		}
	}
	return out
}

// images warns on a content image (one with alt text, outside the page's
// header, footer and navs) that no ImageObject or image URL in the draft
// names, and on an ImageObject for one that lacks its width, height or a
// description.
func (c *checker) images(all []located, p *pageFacts) {
	if p.doc == nil {
		return
	}
	objs := map[string]located{}
	plain := map[string]bool{}
	for _, l := range all {
		if c.is(l.n, "ImageObject") {
			for _, k := range []string{"contentUrl", "url", "thumbnailUrl"} {
				for _, s := range strs(l.n, k) {
					objs[p.abs(s)] = l
				}
			}
			continue
		}
		for _, k := range []string{"image", "logo", "photo", "primaryImageOfPage", "thumbnailUrl"} {
			for _, s := range strs(l.n, k) {
				plain[p.abs(s)] = true
			}
		}
	}
	reported := map[string]bool{}
	visit(p.doc, func(n *xhtml.Node) {
		// a product card's image is its Product's, which the listing check asks for
		if n.DataAtom != atom.Img || chrome(n) || p.inCard(n) {
			return
		}
		alt, _ := attr(n, "alt")
		w, h := attrInt(n, "width"), attrInt(n, "height")
		if strings.TrimSpace(alt) == "" || (w > 0 && w <= 2) || (h > 0 && h <= 2) {
			return
		}
		var obj *located
		covered := false
		for _, u := range p.imgURLs(n) {
			if o, ok := objs[u]; ok {
				obj, covered = &o, true
				break
			}
			covered = covered || plain[u]
		}
		if !covered {
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page's image %q has no ImageObject: add one (contentUrl, width, height, description from its alt text) as the image of the page or of the node it shows", clip(alt, 60))
			return
		}
		if obj == nil || reported[obj.path] {
			return
		}
		var miss []string
		if w > 0 && len(scalars(obj.n, "width")) == 0 {
			miss = append(miss, fmt.Sprintf("width %d", w))
		}
		if h > 0 && len(scalars(obj.n, "height")) == 0 {
			miss = append(miss, fmt.Sprintf("height %d", h))
		}
		if len(strs(obj.n, "description"))+len(strs(obj.n, "caption")) == 0 {
			miss = append(miss, "a description from its alt text")
		}
		if len(miss) > 0 {
			reported[obj.path] = true
			c.add(obj.path, Warning, "this ImageObject is the page's image %q: set %s", clip(alt, 60), strings.Join(miss, ", "))
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
	links := queryAll(n, "a")
	return len(links) == 1 && strings.TrimSpace(htmlText(links[0])) == strings.TrimSpace(htmlText(n))
}

// counts warns on a heading that counts its items ("16 ideas", "Four
// steps") when no ItemList or HowTo in the draft has that many.
func (c *checker) counts(all []located, p *pageFacts) {
	if p.doc == nil {
		return
	}
	have := map[int]bool{}
	for _, l := range all {
		for _, s := range scalars(l.n, "numberOfItems") {
			if i, err := strconv.Atoi(s); err == nil {
				have[i] = true
			}
		}
		if c.is(l.n, "ItemList") || c.is(l.n, "HowTo") {
			for _, k := range []string{"itemListElement", "step"} {
				if raw, ok := l.n.get(k); ok {
					have[len(values(raw))] = true
				}
			}
		}
	}
	visit(p.doc, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4:
		default:
			return
		}
		if chrome(n) || linkHeading(n) || p.inCard(n) {
			return
		}
		t := htmlText(n)
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
			c.add("html:"+selectorFor(p.doc, n), Warning, "the heading %q counts %d items: add an ItemList of them (numberOfItems %d, a ListItem each) or a HowTo with %d steps, in the WebPage's hasPart", clip(t, 60), k, k, k)
			return
		}
	})
}

// sectionCounts are the counts of the items a heading's section holds: its
// subheadings, and the items of the first list, the details and the table
// rows before them. A heading whose number the section does not hold counts
// something else: "2 movers", "two bedrooms", "10 business days".
func sectionCounts(h *xhtml.Node) []int {
	level := headingLevel(h)
	var subs, details, rows, items int
	listDone, before, started := false, true, false
	stop := false
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil && !stop; c = c.NextSibling {
			if skipped(c) || c.Type != xhtml.ElementNode {
				continue
			}
			if c == h {
				started = true
				continue
			}
			if started {
				if l := headingLevel(c); l > 0 {
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
							items, listDone = len(queryAll(c, "li")), true
						}
					case atom.Details:
						details++
					case atom.Tr:
						if len(queryAll(c, "td")) > 0 {
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

// headingLevel is an h1 to h6's level, or 0.
func headingLevel(n *xhtml.Node) int {
	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		return int(n.Data[1] - '0')
	}
	return 0
}

// idBox is the nearest ancestor with an id, or the body.
func idBox(n *xhtml.Node) *xhtml.Node {
	for e := n.Parent; e != nil; e = e.Parent {
		if id, ok := attr(e, "id"); (ok && cssIdent.MatchString(id)) || e.DataAtom == atom.Body {
			return e
		}
	}
	return n.Parent
}

// outsideLinks warns on a group of three or more links to other sites, under
// one ancestor with an id, whose URLs no node in the draft names: partners,
// vendors or sources the page features.
func (c *checker) outsideLinks(p *pageFacts, d *draftIndex) {
	if p.doc == nil || p.base == nil {
		return
	}
	type group struct{ urls, labels []string }
	var order []*xhtml.Node
	groups := map[*xhtml.Node]*group{}
	visit(p.doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.A {
			return
		}
		h, _ := attr(n, "href")
		u, err := url.Parse(strings.TrimSpace(h))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || strings.EqualFold(u.Hostname(), p.base.Hostname()) {
			return
		}
		// the business's profiles and share buttons have checks of their own
		if socialProfile(h) || shareLink.MatchString(h) {
			return
		}
		key := normURL(u)
		if d.urls[key] {
			return
		}
		box := idBox(n)
		g := groups[box]
		if g == nil {
			g = &group{}
			groups[box] = g
			order = append(order, box)
		}
		if !slices.Contains(g.urls, key) {
			g.urls = append(g.urls, key)
			label := strings.TrimSpace(htmlText(n))
			if label == "" {
				label, _ = attr(n, "aria-label")
			}
			if label = clip(label, 40); label != "" && !slices.Contains(g.labels, label) {
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
		c.add("html:"+selectorFor(p.doc, box), Warning, "%d links to outside sites here (%s) name no node: when they are businesses or people the page features, add an ItemList of them to the WebPage's hasPart, each with url and sameAs; when they are sources, list them as citation", len(g.urls), strings.Join(labels, ", "))
	}
}

var (
	relatedLabel = regexp.MustCompile(`(?i)^(?:related\b|keep reading|read next|read more|more (?:for|from|on|like this|guides|articles|reading)|you (?:may|might) also like|see also|further reading|in this (?:guide|section|series|hub)\b|guides in\b)`)
	guideLabel   = regexp.MustCompile(`(?i)^(?:in this (?:guide|section|series|hub)\b|guides in\b)`)
)

// related warns when a block the page labels as related reading links to
// pages of the site that the WebPage's relatedLink does not list. A block of
// product cards is the Product's related products instead.
func (c *checker) related(all []located, p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	var rel []string
	for _, l := range all {
		if c.is(l.n, "WebPage") {
			for _, s := range strs(l.n, "relatedLink") {
				rel = append(rel, p.abs(s))
			}
		}
	}
	for _, b := range p.relatedBoxes() {
		t, box := b.label, b.n
		var cards []*card
		for _, cd := range p.productCards() {
			if cd.related && slices.Contains(ancestors(cd.n), box) {
				cards = append(cards, cd)
			}
		}
		if len(cards) > 0 {
			var names []string
			for _, cd := range cards {
				if !d.urls[cd.url] && !slices.Contains(names, cd.name) {
					names = append(names, cd.name)
				}
			}
			if len(names) > 0 {
				c.add("html:"+selectorFor(p.doc, box), Warning, "the page's %q block lists products (%s) that the Product's isRelatedTo and isSimilarTo do not name: add each as a Product (name, url, image, offers)", t, strings.Join(names, ", "))
			}
			continue
		}
		// a guide's own pages, listed on its hub, are a part of the page
		if guideLabel.MatchString(t) {
			var pages []string
			for _, u := range p.siteLinks(box) {
				if !slices.Contains(pages, u) {
					pages = append(pages, u)
				}
			}
			if len(pages) >= 3 && slices.ContainsFunc(pages, func(u string) bool { return !d.urls[u] }) {
				c.add("html:"+selectorFor(p.doc, box), Warning, "the page's %q block lists %d pages of the site: add an ItemList of them (a ListItem for each with position, url and name) to the WebPage's hasPart; a hub page whose purpose is the list is a CollectionPage with the ItemList as its mainEntity", t, len(pages))
			}
			continue
		}
		var missing []string
		visit(box, func(a *xhtml.Node) {
			if a.DataAtom != atom.A {
				return
			}
			h, _ := attr(a, "href")
			pu, err := url.Parse(strings.TrimSpace(h))
			if h == "" || strings.HasPrefix(h, "#") || err != nil {
				return
			}
			if p.base != nil {
				pu = p.base.ResolveReference(pu)
			}
			if (p.base != nil && !strings.EqualFold(pu.Hostname(), p.base.Hostname())) || (p.base == nil && pu.IsAbs()) {
				return
			}
			if link := pu.String(); !slices.Contains(rel, normURL(pu)) && !slices.Contains(missing, link) {
				missing = append(missing, link)
			}
		})
		if len(missing) > 0 {
			c.add("html:"+selectorFor(p.doc, box), Warning, "the page's %q links are not in the WebPage's relatedLink: add %s", t, strings.Join(missing, ", "))
		}
	}
}

// ancestors are n and the elements around it.
func ancestors(n *xhtml.Node) []*xhtml.Node {
	var out []*xhtml.Node
	for e := n; e != nil; e = e.Parent {
		out = append(out, e)
	}
	return out
}

var responseTime = regexp.MustCompile(`(?i)\b(?:repl(?:y|ies)|responds?|responses?|get back to you|hear back)\b[^.!?]{0,40}?\bwithin\s+(?:(?:\d+|one|two|three)\s+)?(?:business days?|working days?|days?|hours?|minutes?)`)

// responseTime warns when the page promises a reply time and no
// ServiceChannel has it as processingTime.
func (c *checker) responseTime(all []located, p *pageFacts) {
	m := responseTime.FindString(p.text)
	if m == "" || slices.ContainsFunc(all, func(l located) bool { return len(scalars(l.n, "processingTime")) > 0 }) {
		return
	}
	path := pagePath
	if i := slices.IndexFunc(all, func(l located) bool { return c.is(l.n, "ServiceChannel") }); i >= 0 {
		path = all[i].path
	}
	c.add(path, Warning, "the page promises %q but no ServiceChannel has processingTime: set it as a Duration (P1D for a day) on the channel the promise is for", m)
}

var (
	priceAmount = regexp.MustCompile(`(?:A\$|AU\$|AUD ?|US\$|USD ?|NZ\$|NZD ?|C\$|CA\$|S\$|HK\$|€|£|¥|₹|\$)\s?(\d{1,3}(?:,\d{3})+|\d+)(?:\.(\d{1,2}))?`)
	taxWords    = regexp.MustCompile(`(?i)\b(plus|excl(?:uding|\.)?|ex|exclusive of|incl(?:uding|\.)?|inc\.?|inclusive of)\s+(GST|VAT|tax(?:es)?|sales tax)\b|\b(GST|VAT|tax(?:es)?)\s+(included|inclusive|incl\.?|excluded|exclusive|excl\.?|extra)\b`)
	taxOut      = regexp.MustCompile(`(?i)^(?:plus|ex|excl|exclusive|excluded|extra)`)
	notPrices   = regexp.MustCompile(`(?i)\b(?:save|payments? of|under|below|less than)\s+(?:A\$|AU\$|\$|£|€)\s?\d[\d,]*(?:\.\d{1,2})?`)
)

// amount reads a price's number.
func amount(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", ""), 64)
	return f, err == nil
}

// prices warns on prices the page shows that no price property carries, on
// a price without its currency, and on tax wording the PriceSpecification
// does not match.
func (c *checker) prices(all []located, p *pageFacts) {
	// marked holds every amount the draft carries; sold, only the prices of
	// things for sale, which tax wording is about (a shipping rate's
	// MonetaryAmount is not one)
	marked, sold := map[float64]bool{}, map[float64]bool{}
	for _, l := range all {
		keys := []string{"price", "lowPrice", "highPrice", "minPrice", "maxPrice"}
		if c.is(l.n, "MonetaryAmount") || c.is(l.n, "PriceSpecification") {
			keys = append(keys, "value", "minValue", "maxValue")
		}
		priced := false
		for i, k := range keys {
			for _, s := range scalars(l.n, k) {
				if f, ok := amount(s); ok {
					marked[f], priced = true, true
					sold[f] = sold[f] || i < 5
				}
			}
		}
		if priced && len(strs(l.n, "priceCurrency"))+len(strs(l.n, "currency")) == 0 {
			c.add(l.path, Warning, "this %s has a price but no priceCurrency: set it (AUD, USD, EUR)", typeNames(l.n)[0])
		}
	}
	// prices in the site's header, footer and menus, a saving ("save $50")
	// and an instalment are no item's price; the shop checks report the
	// prices they own (was prices, shipping rates, product cards)
	ms := priceAmount.FindAllStringSubmatch(notPrices.ReplaceAllString(p.main, " "), -1)
	if len(ms) == 0 {
		return
	}
	var missing []string
	seen := map[float64]bool{}
	anySold := false
	for _, m := range ms {
		f, ok := priceOf(m)
		if !ok || seen[f] {
			continue
		}
		seen[f] = true
		anySold = anySold || sold[f]
		if !marked[f] && !p.claimed[f] {
			missing = append(missing, strings.TrimSpace(m[0]))
		}
	}
	switch {
	case len(missing) == 0:
	case p.sells():
		c.add(pagePath, Warning, "the page shows prices (%s) that no price property carries: mark each up on the item it is the price of, as an Offer's price or, for a range, an AggregateOffer's lowPrice and highPrice, with priceCurrency and offerCount", strings.Join(missing, ", "))
	default:
		// a site that sells nothing shows what others charge
		c.add(pagePath, Warning, "the page shows prices (%s) that no price property carries: mark each up on the thing it is the price of, as a MonetaryAmount (value, or minValue and maxValue, currency) for a cost or estimate of what others charge, or an Offer's price only for what the business itself sells", strings.Join(missing, ", "))
	}
	t := taxWords.FindStringSubmatch(p.text)
	if t == nil || !anySold {
		return
	}
	incl := !taxOut.MatchString(firstNonEmpty(t[1], t[4]))
	found := false
	for _, l := range all {
		for _, v := range scalars(l.n, "valueAddedTaxIncluded") {
			found = true
			if (v == "true") != incl {
				c.add(l.path+".valueAddedTaxIncluded", Warning, "valueAddedTaxIncluded is %s but the page says %q", v, t[0])
			}
		}
	}
	if !found {
		c.add(pagePath, Warning, "the page's prices are %q but no PriceSpecification sets valueAddedTaxIncluded: set it %t", t[0], incl)
	}
}

// attribution is who a blockquote is from: a cite inside it, or after it in
// its figure.
func attribution(n *xhtml.Node) string {
	for e := n; e != nil && e.DataAtom != atom.Body; e = e.Parent {
		if cs := queryAll(e, "cite"); len(cs) > 0 {
			return " (" + strings.TrimSpace(htmlText(cs[0])) + ")"
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
func (c *checker) quotes(p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	var carried []string
	exact := map[string]bool{}
	for _, s := range d.strs {
		w := normWords(stripTags(s))
		exact[w] = true
		if strings.Count(w, " ") >= 2 {
			carried = append(carried, " "+w+" ")
		}
	}
	visit(p.doc, func(n *xhtml.Node) {
		text, who := htmlText(n), ""
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
		q := normWords(text)
		if q == "" || exact[q] || slices.ContainsFunc(carried, func(s string) bool { return strings.Contains(s, " "+q+" ") }) {
			return
		}
		c.add("html:"+selectorFor(p.doc, n), Warning, "the page quotes %q%s and no node carries it: a customer's opinion of a business, product or service is a Review of it (reviewBody, author, itemReviewed); another quotation is a Quotation (text, creator)", clip(text, 60), who)
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
	t := strings.TrimSpace(htmlText(n))
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
		cites := queryAll(n, "cite")
		if n.Parent != nil {
			for ch := n.Parent.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.DataAtom == atom.Cite {
					cites = append(cites, ch)
				}
			}
		}
		switch {
		case len(cites) > 0:
			who = strings.TrimSpace(htmlText(cites[0]))
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
func (c *checker) ratings(all []located, p *pageFacts) {
	var shownVals [][]string
	for _, loc := range ratingShown.FindAllStringSubmatchIndex(p.text, -1) {
		m := p.text[loc[0]:loc[1]]
		// "1 / 10" with no rating word before it is a gallery's counter
		if integerOutOf.MatchString(m) && !ratingWord.MatchString(p.text[max(0, loc[0]-30):loc[0]]) {
			continue
		}
		shownVals = append(shownVals, []string{m, p.text[loc[2]:loc[3]]})
	}
	if p.doc != nil {
		visit(p.doc, func(n *xhtml.Node) {
			for _, k := range []string{"aria-label", "title"} {
				if v, ok := attr(n, k); ok {
					shownVals = append(shownVals, ratingShown.FindAllStringSubmatch(v, -1)...)
				}
			}
		})
	}
	have := map[float64]bool{}
	for _, l := range all {
		if c.is(l.n, "Rating") {
			for _, s := range scalars(l.n, "ratingValue") {
				if f, ok := amount(s); ok {
					have[f] = true
				}
			}
		}
	}
	var missing []string
	for _, m := range shownVals {
		if f, ok := amount(m[1]); ok && !have[f] && !slices.Contains(missing, m[1]) {
			missing = append(missing, m[1])
		}
	}
	if len(missing) > 0 {
		c.add(pagePath, Warning, "the page shows ratings (%s) that no Rating or AggregateRating carries as ratingValue: mark each up on the Review or item it rates (ratingValue, bestRating, and ratingCount for an aggregate)", strings.Join(missing, ", "))
	}
}

// offers warns on an Offer that names no seller. An AggregateOffer gathers
// many sellers' offers, so it needs none.
func (c *checker) offers(all []located) {
	for _, l := range all {
		if !c.is(l.n, "Offer") || c.is(l.n, "AggregateOffer") {
			continue
		}
		_, s := l.n.get("seller")
		_, o := l.n.get("offeredBy")
		if !s && !o {
			c.add(l.path, Warning, "this Offer names no seller: set seller (or offeredBy) to the business that makes it")
		}
	}
}

var personName = regexp.MustCompile(`^(\p{Lu}[\p{L}'’-]+)(?:\s+\p{Lu}[\p{L}'’-]+)?\s+(\p{Lu}[\p{L}'’-]+)$`)

// personNames warns on a Person named with a given and a family name that
// does not give them apart.
func (c *checker) personNames(all []located) {
	for _, l := range all {
		if !c.is(l.n, "Person") {
			continue
		}
		_, g := l.n.get("givenName")
		_, f := l.n.get("familyName")
		for _, s := range strs(l.n, "name") {
			if m := personName.FindStringSubmatch(strings.TrimSpace(s)); m != nil && !g && !f {
				c.add(l.path, Warning, "this Person is named %q: set givenName %s and familyName %s", s, m[1], m[2])
			}
			break
		}
	}
}

// pageNames warns on a WebPage node, not a reference, that has no url or no
// name.
func (c *checker) pageNames(all []located) {
	for _, l := range all {
		if !c.is(l.n, "WebPage") {
			continue
		}
		var miss []string
		for _, k := range []string{"url", "name"} {
			if _, ok := l.n.get(k); !ok {
				miss = append(miss, k)
			}
		}
		if len(miss) > 0 {
			c.add(l.path, Warning, "this %s has no %s: give it the page's", typeNames(l.n)[0], strings.Join(miss, " or "))
		}
	}
}

// bareTypes name nothing more than a category; a node is one of their
// subtypes.
var bareTypes = map[string]string{
	"Intangible":      "Service, Offer, DefinedTerm, Audience",
	"StructuredValue": "PropertyValue, QuantitativeValue, MonetaryAmount, ContactPoint",
	"Action":          "AskAction, ReadAction, SearchAction, QuoteAction",
	"MediaObject":     "ImageObject, VideoObject, AudioObject",
	"Enumeration":     "a member of the enumeration",
}

// bare warns on a node typed only as one of the bare types.
func (c *checker) bare(all []located) {
	for _, l := range all {
		if types := typeNames(l.n); len(types) == 1 {
			if use, ok := bareTypes[types[0]]; ok {
				c.add(l.path, Warning, "a bare %s says little: use the subtype that fits (%s)", types[0], use)
			}
		}
	}
}

// ownReviews notes a Review or AggregateRating of the page's own publisher
// or provider: valid markup, but Google shows no stars for it.
func (c *checker) ownReviews(all []located) {
	own := map[string]bool{}
	for _, l := range all {
		for _, k := range []string{"publisher", "provider"} {
			raw, _ := l.n.get(k)
			for _, v := range values(raw) {
				switch x := v.(type) {
				case *object:
					for _, id := range strs(x, "@id") {
						own[id] = true
					}
				case string:
					own[x] = true
				}
			}
		}
	}
	isOwn := func(v any) bool {
		if o, ok := v.(*object); ok {
			return slices.ContainsFunc(strs(o, "@id"), func(id string) bool { return own[id] })
		}
		s, _ := v.(string)
		return own[s]
	}
	for _, l := range all {
		if c.is(l.n, "Review") || c.is(l.n, "AggregateRating") {
			raw, _ := l.n.get("itemReviewed")
			if slices.ContainsFunc(values(raw), isOwn) {
				c.add(l.path, Info, "a %s of the page's own business or person: the markup is valid, but Google shows no review stars for self-serving reviews", typeNames(l.n)[0])
			}
		}
		if slices.ContainsFunc(strs(l.n, "@id"), func(id string) bool { return own[id] }) {
			for _, k := range []string{"review", "aggregateRating"} {
				if _, ok := l.n.get(k); ok {
					c.add(l.path+"."+k, Info, "the page's own business or person carries %s: the markup is valid, but Google shows no review stars for self-serving reviews", k)
				}
			}
		}
	}
}

var reviewNested = regexp.MustCompile(`\.review(?:\[\d+\])?$`)

// reviews warns on a Review without an author, or without the item it
// reviews when it is not nested in that item's review.
func (c *checker) reviews(all []located) {
	for _, l := range all {
		if !c.is(l.n, "Review") || len(l.n.keys) < 2 {
			continue
		}
		if _, ok := l.n.get("author"); !ok {
			c.add(l.path, Warning, "this Review has no author: name who wrote it")
		}
		if _, ok := l.n.get("itemReviewed"); !ok && !reviewNested.MatchString(l.path) {
			c.add(l.path, Warning, "this Review has no itemReviewed: name what it reviews, or put it in that item's review")
		}
	}
}

// numericOnly reports a property whose ranges are all numbers.
func numericOnly(ranges []string) bool {
	return len(ranges) > 0 && !slices.ContainsFunc(ranges, func(r string) bool { return r != "Number" && r != "Integer" && r != "Float" })
}
