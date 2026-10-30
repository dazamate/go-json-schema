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
	doc  *xhtml.Node // nil when only the page's text is known
	text string      // the visible text, whitespace collapsed
	norm string      // normWords(text), padded with spaces
	base *url.URL    // the page's canonical URL, or nil
}

func newPageFacts(doc *xhtml.Node, text string) *pageFacts {
	p := &pageFacts{doc: doc, text: text, norm: " " + normWords(text) + " "}
	if doc != nil {
		p.base = canonical(doc)
	}
	return p
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
	return slices.ContainsFunc(typeNames(n), func(t string) bool { return slices.Contains(c.t.v.Ancestors(t), ty) })
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

// content runs the checks that compare the draft with the page.
func (c *checker) content(all []located, p *pageFacts) {
	d := indexDraft(all)
	c.shownText(all, p)
	c.anchors(all, p)
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
	c.related(all, p)
	c.responseTime(all, p)
	c.prices(all, p)
	c.quotes(p, d)
	c.ratings(all, p)
	c.offers(all)
	c.personNames(all)
	c.pageNames(all)
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

// search warns on a search box with no SearchAction in the draft.
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
		if role, _ := attr(n, "role"); role == "search" || len(queryAll(n, `input[type="search"]`)) > 0 {
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
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page's form has no Action whose target points at it: add the Action it performs (AskAction, QuoteAction, ReserveAction, SubscribeAction and the like) %s", where)
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
		visit(p.doc, func(n *xhtml.Node) {
			if n.DataAtom == atom.Dfn || (n.DataAtom == atom.Dt && n.Parent != nil && n.Parent.DataAtom == atom.Dl) {
				add(htmlText(n))
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
		if n.DataAtom != atom.Img || chrome(n) {
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
		"guests": true, "people": true, "dollars": true, "cents": true, "percent": true, "times": true, "metres": true, "meters": true, "kilometres": true, "kilometers": true, "kms": true, "miles": true, "stars": true, "points": true}
)

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
		if chrome(n) {
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
			if k < 2 || have[k] {
				continue
			}
			c.add("html:"+selectorFor(p.doc, n), Warning, "the heading %q counts %d items: add an ItemList of them (numberOfItems %d, a ListItem each) or a HowTo with %d steps, in the WebPage's hasPart", clip(t, 60), k, k, k)
			return
		}
	})
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

var relatedLabel = regexp.MustCompile(`(?i)^(?:related\b|keep reading|read next|read more|more (?:for|from|on|like this|guides|articles|reading)|you (?:may|might) also like|see also|further reading)`)

// related warns when a block the page labels as related reading links to
// pages of the site that the WebPage's relatedLink does not list.
func (c *checker) related(all []located, p *pageFacts) {
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
	done := map[*xhtml.Node]bool{}
	visit(p.doc, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.P, atom.Span:
		default:
			return
		}
		t := strings.TrimSpace(htmlText(n))
		if len(strings.Fields(t)) > 6 || !relatedLabel.MatchString(t) || chrome(n) {
			return
		}
		box := n.Parent
		for e := n.Parent; e != nil; e = e.Parent {
			if e.DataAtom == atom.Section || e.DataAtom == atom.Aside || e.DataAtom == atom.Nav || e.DataAtom == atom.Article {
				box = e
				break
			}
		}
		if box == nil || done[box] {
			return
		}
		done[box] = true
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
	})
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
	taxWords    = regexp.MustCompile(`(?i)\b(plus|excl(?:uding|\.)?|ex|exclusive of|incl(?:uding|\.)?|inc\.?|inclusive of)\s+(GST|VAT|tax(?:es)?|sales tax)\b`)
	taxOut      = regexp.MustCompile(`(?i)^(?:plus|ex)`)
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
	marked := map[float64]bool{}
	for _, l := range all {
		keys := []string{"price", "lowPrice", "highPrice", "minPrice", "maxPrice"}
		if c.is(l.n, "MonetaryAmount") || c.is(l.n, "PriceSpecification") {
			keys = append(keys, "value", "minValue", "maxValue")
		}
		priced := false
		for _, k := range keys {
			for _, s := range scalars(l.n, k) {
				if f, ok := amount(s); ok {
					marked[f], priced = true, true
				}
			}
		}
		if priced && len(strs(l.n, "priceCurrency"))+len(strs(l.n, "currency")) == 0 {
			c.add(l.path, Warning, "this %s has a price but no priceCurrency: set it (AUD, USD, EUR)", typeNames(l.n)[0])
		}
	}
	ms := priceAmount.FindAllStringSubmatch(p.text, -1)
	if len(ms) == 0 {
		return
	}
	var missing []string
	seen := map[float64]bool{}
	for _, m := range ms {
		f, ok := amount(m[1] + "." + m[2])
		if m[2] == "" {
			f, ok = amount(m[1])
		}
		if !ok || seen[f] {
			continue
		}
		seen[f] = true
		if !marked[f] {
			missing = append(missing, strings.TrimSpace(m[0]))
		}
	}
	if len(missing) > 0 {
		c.add(pagePath, Warning, "the page shows prices (%s) that no price property carries: mark each up on the item it is the price of, as an Offer's price or, for a range, an AggregateOffer's lowPrice and highPrice, with priceCurrency", strings.Join(missing, ", "))
	}
	t := taxWords.FindStringSubmatch(p.text)
	if t == nil || len(missing) == len(seen) {
		return
	}
	incl := !taxOut.MatchString(t[1])
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
// carries: a testimonial is a Review, another quote a Quotation.
func (c *checker) quotes(p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	var carried []string
	for _, s := range d.strs {
		if w := normWords(stripTags(s)); strings.Count(w, " ") >= 2 {
			carried = append(carried, " "+w+" ")
		}
	}
	visit(p.doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.Blockquote {
			return
		}
		q := normWords(htmlText(n))
		if q == "" || slices.ContainsFunc(carried, func(s string) bool { return strings.Contains(s, " "+q+" ") }) {
			return
		}
		c.add("html:"+selectorFor(p.doc, n), Warning, "the page quotes %q%s and no node carries it: a customer's opinion of a business, product or service is a Review of it (reviewBody, author, itemReviewed); another quotation is a Quotation (text, creator)", clip(htmlText(n), 60), attribution(n))
	})
}

var ratingShown = regexp.MustCompile(`(?i)\b(\d(?:\.\d)?)\s*(?:/\s*(?:5|10)\b|out of\s*(?:5|10)\b|stars?\b)`)

// ratings warns on ratings the page shows that no Rating carries as its
// ratingValue.
func (c *checker) ratings(all []located, p *pageFacts) {
	shownVals := ratingShown.FindAllStringSubmatch(p.text, -1)
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
