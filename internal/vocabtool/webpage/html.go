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
// scripts, styles, noscript, template contents and hidden elements.
func Skipped(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Noscript, atom.Template:
		return true
	}
	_, hidden := Attr(n, "hidden")
	aria, _ := Attr(n, "aria-hidden")
	return (hidden && !streamed(n)) || aria == "true"
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

// scoped reports a header or footer that belongs to a part of the page. One
// inside main is the site's own when the page wraps everything in main: a
// header that links to the home page and holds no h1, or a footer with a
// copyright line.
func scoped(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		switch p.DataAtom {
		case atom.Article, atom.Aside, atom.Nav, atom.Section:
			return true
		case atom.Main:
			return !siteChrome(n)
		}
	}
	return false
}

// siteChrome reports a header or footer that is the site's, not its section's.
func siteChrome(n *html.Node) bool {
	switch n.DataAtom {
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
		return CopyrightLine.MatchString(HTMLText(n))
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

// CopyrightLine matches a copyright line and captures its last year.
var CopyrightLine = regexp.MustCompile(`(?i)(?:©|\(c\)|copyright)\s*(?:\d{4}\s*[-–]\s*)?(\d{4})`)

// Chrome reports an element in the page's own header, footer or a nav.
func Chrome(n *html.Node) bool {
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
func siteChromeAround(n *html.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if (e.DataAtom == atom.Header || e.DataAtom == atom.Footer) && !scoped(e) {
			return true
		}
	}
	return false
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
