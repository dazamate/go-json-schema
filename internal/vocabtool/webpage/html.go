package webpage

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var reactSegment = regexp.MustCompile(`^S:\d+$`)

// Skipped reports whether a visitor never sees the element or its contents:
// scripts, styles, noscript, template contents, hidden elements, and the
// title and desc of an icon in a link or button, which name the icon for a
// screen reader and show as no text (a menu link's "Nav chevron").
func Skipped(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if n.Namespace == "svg" && (n.Data == "title" || n.Data == "desc") && inControl(n) {
		return true
	}
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Noscript, atom.Template:
		return true
	}
	_, hidden := Attr(n, "hidden")
	aria, _ := Attr(n, "aria-hidden")
	return (hidden && !streamed(n)) || aria == "true"
}

// inControl reports an element in a link, button, label or summary, or in an
// element with a link, button, menu item or tab role. A graphic there is the
// control's icon; a graphic outside one (a card logo in a row of ways to pay,
// a chart) is the page's own, and its title says what it shows.
func inControl(n *html.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		switch e.DataAtom {
		case atom.A, atom.Button, atom.Label, atom.Summary:
			return true
		}
		switch role, _ := Attr(e, "role"); role {
		case "button", "link", "menuitem", "tab":
			return true
		}
	}
	return false
}

// streamed reports a hidden div React's streaming server render sends a
// part of the page in (id "S:0"), which its script moves into place: the
// served HTML hides it, the rendered page shows it.
func streamed(n *html.Node) bool {
	id, _ := Attr(n, "id")
	return n.DataAtom == atom.Div && reactSegment.MatchString(id)
}

// Attr is an attribute's value and whether the element has it.
func Attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// Visit calls fn on every element a visitor can see, in document order.
func Visit(n *html.Node, fn func(*html.Node)) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if Skipped(c) {
			continue
		}
		if c.Type == html.ElementNode {
			fn(c)
		}
		Visit(c, fn)
	}
}

// HTMLText is the visible text under n: every text node outside the skipped
// elements, whitespace collapsed.
func HTMLText(doc *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if Skipped(c) {
				continue
			}
			if c.Type == html.TextNode {
				b.WriteString(c.Data)
				b.WriteByte(' ')
			}
			walk(c)
		}
	}
	walk(doc)
	return strings.Join(strings.Fields(b.String()), " ")
}

// iconText is the text under n with each inline svg read as its name, its
// aria-label or else its title: what a screen reader calls a control whose
// only label is its icon (a cart link's "Open Cart Menu").
func iconText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c.Type == html.ElementNode && c.Namespace == "svg" && c.Data == "svg" && !Skipped(c):
				name, _ := Attr(c, "aria-label")
				if strings.TrimSpace(name) == "" {
					for t := c.FirstChild; t != nil; t = t.NextSibling {
						if t.Type == html.ElementNode && t.Data == "title" {
							name = textOf(t)
						}
					}
				}
				b.WriteString(name)
				b.WriteByte(' ')
			case Skipped(c):
			case c.Type == html.TextNode:
				b.WriteString(c.Data)
				b.WriteByte(' ')
			default:
				walk(c)
			}
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// textOf is every text node under n, hidden or not.
func textOf(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
		b.WriteString(textOf(c))
	}
	return b.String()
}

// scoped reports a header or footer that belongs to a part of the page, or to
// the site's header or footer around it. One inside main is the site's own
// when the page wraps everything in main: a header that links to the home
// page and holds no h1, or a footer with a copyright line.
func scoped(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if landmark(p) != 0 {
			return true
		}
		switch p.DataAtom {
		case atom.Article, atom.Aside, atom.Section:
			return true
		case atom.Main:
			return !siteChrome(n)
		}
	}
	return false
}

// siteChrome reports a header or footer that is the site's, not its section's.
func siteChrome(n *html.Node) bool {
	switch landmark(n) {
	case atom.Header:
		if len(cascadia.QueryAll(n, cascadia.MustCompile("h1"))) > 0 {
			return false
		}
		for _, a := range cascadia.QueryAll(n, cascadia.MustCompile("a[href]")) {
			h, _ := Attr(a, "href")
			if u, err := url.Parse(strings.TrimSpace(h)); err == nil && h != "" && (u.Host == "" || u.Path == "" || u.Path == "/") {
				if segs := Segments(u); len(segs) == 0 || (len(segs) == 1 && LocaleSegment.MatchString(segs[0])) {
					return true
				}
			}
		}
	case atom.Footer:
		t := HTMLText(n)
		return CopyrightLine.MatchString(t) || CopyrightYearFirst.MatchString(t)
	}
	return false
}

var (
	// overlayWord is a word of a tag, class or id that names an overlay the
	// page opens over itself: a drawer, modal, dialog or popup.
	overlayWord = regexp.MustCompile(`^(?:drawers?|modal|dialog|popup|popover|lightbox|flyout)$`)
	// controlWord is a word of a class or id that names what opens or closes
	// an overlay, not the overlay ("review-modal-trigger", "dialog-link").
	controlWord = regexp.MustCompile(`^(?:trigger|toggle|toggler|link|btn|button|open|opener)$`)
)

// Overlay reports an element in a drawer, modal or dialog the page opens over
// itself (a cart drawer, a help modal, an off-canvas mobile menu): an
// OverlayBox or an element inside one.
func Overlay(n *html.Node) bool {
	for e := n; e != nil && e.Type == html.ElementNode; e = e.Parent {
		switch e.DataAtom {
		case atom.Body, atom.Html, atom.Main:
			return false
		}
		if OverlayBox(e) {
			return true
		}
	}
	return false
}

// OverlayBox reports a drawer, modal or dialog the page opens over itself: a
// dialog element or role, an aria-modal, or a tag, class or id that names
// one. A link or button that opens one is a control on the page, and an
// element that holds the page's main is the page, whatever their classes
// say.
func OverlayBox(e *html.Node) bool {
	if e.Type != html.ElementNode {
		return false
	}
	switch e.DataAtom {
	case atom.Body, atom.Html, atom.Main, atom.A, atom.Button:
		return false
	case atom.Dialog:
		return true
	}
	role, _ := Attr(e, "role")
	modal, _ := Attr(e, "aria-modal")
	switch {
	case role == "button" || role == "link":
		return false
	case role == "dialog" || role == "alertdialog" || modal == "true":
		return true
	}
	return namesOverlay(e) && !holds(e, atom.Main)
}

// namesOverlay reports an element whose tag, class or id names an overlay
// (site-drawer, "drw-Drawer", "modal fade", "oddit-cart-drawer") or an
// off-canvas mobile menu ("ow-mobile-menu", "mobile-nav"), not a control
// for one.
func namesOverlay(e *html.Node) bool {
	id, _ := Attr(e, "id")
	class, _ := Attr(e, "class")
	for _, s := range append(strings.Fields(class), e.Data, id) {
		words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return r == '-' || r == '_' })
		if slices.ContainsFunc(words, controlWord.MatchString) {
			continue
		}
		for i, w := range words {
			if overlayWord.MatchString(w) || (w == "mobile" && i+1 < len(words) && (words[i+1] == "menu" || words[i+1] == "nav")) {
				return true
			}
		}
	}
	return false
}

// policyWord is a word that names a shop's policy: its shipping, returns or
// guarantee.
var policyWord = regexp.MustCompile(`^(?:shipping|delivery|postage|returns?|refunds?|exchanges?|guarantee|warranty)$`)

// PolicyText is the visible text of the overlays the page opens over itself
// for its shipping, returns or guarantee terms (taylorstitch.com's
// dialog#shipping-dialog, a "Shipping & Returns" drawer): Main leaves
// overlays out, but these terms are the shop's own. A cart, wishlist or help
// overlay gives none.
func (p *Facts) PolicyText() string {
	if p.Doc == nil {
		return ""
	}
	var out []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if OverlayBox(c) && namesPolicy(c) {
				out = append(out, HTMLText(c))
			} else {
				walk(c)
			}
		}
	}
	walk(p.Doc)
	return strings.Join(out, "\n")
}

// namesPolicy reports an overlay whose id, class, label or first heading
// names a shop's policy.
func namesPolicy(e *html.Node) bool {
	id, _ := Attr(e, "id")
	class, _ := Attr(e, "class")
	label, _ := Attr(e, "aria-label")
	names := []string{id, class, label}
	var heading *html.Node
	Visit(e, func(n *html.Node) {
		if heading == nil && HeadingLevel(n) > 0 {
			heading = n
		}
	})
	if heading != nil {
		names = append(names, HTMLText(heading))
	}
	for _, s := range names {
		words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return r < 'a' || r > 'z' })
		if slices.ContainsFunc(words, policyWord.MatchString) {
			return true
		}
	}
	return false
}

// hiddenStyle matches an inline style that hides its element.
var hiddenStyle = regexp.MustCompile(`(?i)(?:^|;)\s*(?:display\s*:\s*none|visibility\s*:\s*hidden)\b`)

// StyleHidden reports an element that its own or an ancestor's inline style
// hides (display: none): a closed drawer, a helper frame. Its text still
// counts as the page's (a closed accordion's answer), but it is no section
// of the page.
func StyleHidden(n *html.Node) bool {
	for e := n; e != nil; e = e.Parent {
		if s, _ := Attr(e, "style"); hiddenStyle.MatchString(s) {
			return true
		}
	}
	return false
}

// CSSIdent matches a name usable in a CSS selector as an id.
var CSSIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// SelectorFor is a CSS selector that finds n alone on the page: its id, its
// tag and aria-label when no other element shares them, or else its path of
// child steps from the nearest ancestor with an id, or from body.
func SelectorFor(doc, n *html.Node) string {
	if id, _ := Attr(n, "id"); CSSIdent.MatchString(id) {
		return "#" + id
	}
	if l, _ := Attr(n, "aria-label"); l != "" {
		s := fmt.Sprintf("%s[aria-label=%q]", n.Data, l)
		if g, err := cascadia.ParseGroup(s); err == nil && len(cascadia.QueryAll(doc, g)) == 1 {
			return s
		}
	}
	var steps []string
	for e := n; e != nil && e.Type == html.ElementNode; e = e.Parent {
		if id, _ := Attr(e, "id"); e != n && CSSIdent.MatchString(id) {
			steps = append(steps, "#"+id)
			break
		}
		if e.DataAtom == atom.Body {
			steps = append(steps, "body")
			break
		}
		steps = append(steps, e.Data+nthOfType(e))
	}
	slices.Reverse(steps)
	return strings.Join(steps, " > ")
}

// nthOfType is ":nth-of-type(i)" when the element has siblings of its tag.
func nthOfType(e *html.Node) string {
	i, n := 0, 0
	for c := e.Parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == e.Data {
			n++
			if c == e {
				i = n
			}
		}
	}
	if n < 2 {
		return ""
	}
	return fmt.Sprintf(":nth-of-type(%d)", i)
}

// QueryAll finds the elements under n that the CSS selector matches. The
// selector must be valid.
func QueryAll(n *html.Node, sel string) []*html.Node {
	return cascadia.QueryAll(n, cascadia.MustCompile(sel))
}

// Shown reports whether a visitor can see the element: no ancestor of it is
// Skipped.
func Shown(n *html.Node) bool {
	for e := n; e != nil; e = e.Parent {
		if Skipped(e) {
			return false
		}
	}
	return true
}

// Ancestors are n and the elements around it.
func Ancestors(n *html.Node) []*html.Node {
	var out []*html.Node
	for e := n; e != nil; e = e.Parent {
		out = append(out, e)
	}
	return out
}

// Detail is a labelled value the page lists: a key/value list's term and
// description, a two-column table's row, a "Material: silver" line.
type Detail struct{ Label, Value string }

// Pairs are a key/value list's labels and values, in order. A dt may sit in
// a div of the dl.
func Pairs(dl *html.Node) []Detail {
	var out []Detail
	var label string
	Visit(dl, func(n *html.Node) {
		switch n.DataAtom {
		case atom.Dt:
			label = HTMLText(n)
		case atom.Dd:
			if label != "" {
				out = append(out, Detail{Label: label, Value: HTMLText(n)})
				label = ""
			}
		}
	})
	return out
}

// MustParse parses a URL this package normalized.
func MustParse(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

// CopyrightLine matches a copyright line: the mark, the holder's name when
// it comes before the year ("© Oliver Bonas 2026", "Copyright @ 2026"), and
// the year or the range of years. It captures the range's first year (if
// any) and the last.
var CopyrightLine = regexp.MustCompile(`(?i)(?:©|\(c\)|copyright)[\s@:©.]*(?:[\p{L}&'’.,/ -]{0,50}?\s)?(?:((?:19|20)\d{2})\s*[-–—]\s*)?((?:19|20)\d{2})\b`)

// CopyrightYearFirst matches a copyright line with the year before the mark
// ("2026 © Acme"), and captures the year.
var CopyrightYearFirst = regexp.MustCompile(`(?i)\b((?:19|20)\d{2})\s*(?:©|\(c\))`)

// landmark is the chrome an element is by its tag or its ARIA landmark role:
// atom.Nav for a nav or role navigation, atom.Header for a header or role
// banner, atom.Footer for a footer or role contentinfo, else 0.
func landmark(n *html.Node) atom.Atom {
	switch n.DataAtom {
	case atom.Nav, atom.Header, atom.Footer:
		return n.DataAtom
	}
	switch role, _ := Attr(n, "role"); role {
	case "navigation":
		return atom.Nav
	case "banner":
		return atom.Header
	case "contentinfo":
		return atom.Footer
	}
	return 0
}

// Chrome reports an element in the page's own header, footer or a nav, by
// their tags, their landmark roles or the class or id the site names them
// by, or in a cookie consent block.
func Chrome(n *html.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		switch landmark(e) {
		case atom.Nav:
			return true
		case atom.Header, atom.Footer:
			if !scoped(e) {
				return true
			}
		}
		switch chromeName(e) {
		case chromeNamed:
			if !scoped(e) {
				return true
			}
		case consentNamed:
			return true
		}
	}
	return false
}

// siteChromeAround reports an element in the site's own header or footer.
func siteChromeAround(n *html.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if l := landmark(e); (l == atom.Header || l == atom.Footer || chromeName(e) == chromeNamed) && !scoped(e) {
			return true
		}
	}
	return false
}

var (
	// chromeToken is a whole class or id token a site names its header,
	// footer or menu by ("footer", "site-header", "navbar"), or the section
	// group a Shopify theme puts its header or footer in. A token that only
	// holds the word names a part of the copy: a price list's header, a
	// product description's header, a menu item.
	chromeToken = regexp.MustCompile(`(?i)^(?:footer|site-footer|page-footer|header|site-header|navbar|nav|menu|shopify-section-group-(?:[a-z0-9]+-)*(?:header|footer)(?:-[a-z0-9]+)*)$`)
	// consentToken is in the class or id of a cookie consent block:
	// Cookiebot's CybotCookiebotDialog, OneTrust's onetrust-banner-sdk, a
	// cookie-banner, cookie-consent, cookie-notice, cookie-bar or cookie-law
	// block, CookieYes's cky-consent.
	consentToken = regexp.MustCompile(`(?i)cookiebot|onetrust|cookie-?(?:banner|consent|notice|bar|law)|cky-consent`)
)

// The ways chromeName finds an element named as chrome.
const (
	chromeNamed  = 1 // the site's header, footer or menu
	consentNamed = 2 // a cookie consent block
)

// chromeName is how an element's class or id names it chrome: chromeNamed,
// consentNamed, or 0. The body and an element that holds the page's main
// are the page, whatever their classes say ("gdpr-infobar-visible").
func chromeName(e *html.Node) int {
	if e.Type != html.ElementNode || e.DataAtom == atom.Body || e.DataAtom == atom.Html {
		return 0
	}
	id, _ := Attr(e, "id")
	class, _ := Attr(e, "class")
	found := 0
	for _, s := range append(strings.Fields(class), id) {
		switch t := strings.ReplaceAll(s, "_", "-"); {
		case consentToken.MatchString(t):
			found = consentNamed
		case found == 0 && chromeToken.MatchString(t):
			found = chromeNamed
		}
	}
	if found != 0 && holds(e, atom.Main) {
		return 0
	}
	return found
}

// HeadingLevel is an h1 to h6's level, or 0.
func HeadingLevel(n *html.Node) int {
	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		return int(n.Data[1] - '0')
	}
	return 0
}

// IDBox is the nearest ancestor with an id, or the body.
func IDBox(n *html.Node) *html.Node {
	for e := n.Parent; e != nil; e = e.Parent {
		if id, ok := Attr(e, "id"); (ok && CSSIdent.MatchString(id)) || e.DataAtom == atom.Body {
			return e
		}
	}
	return n.Parent
}

// NextElement is the element after n among its siblings, or nil.
func NextElement(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}

// KeyValueList reports a dl of short values (Material: Silver), not a
// glossary of definitions.
func KeyValueList(dl *html.Node) bool {
	dds := QueryAll(dl, "dd")
	if len(dds) == 0 {
		return false
	}
	long := 0
	for _, dd := range dds {
		t := HTMLText(dd)
		if len(strings.Fields(t)) >= 6 || strings.HasSuffix(t, ".") {
			long++
		}
	}
	return long*2 <= len(dds)
}
