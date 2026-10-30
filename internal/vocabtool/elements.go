package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// elementTypeNames are the types a page section can need: the WebPageElement
// subtypes, and BreadcrumbList for a breadcrumb nav.
var elementTypeNames = []string{"SiteNavigationElement", "WPHeader", "WPFooter", "WPSideBar", "Table", "WPAdBlock", "BreadcrumbList"}

// roleTypes are the ARIA landmark and table roles that name a section type.
var roleTypes = map[string]string{
	"navigation":    "SiteNavigationElement",
	"banner":        "WPHeader",
	"contentinfo":   "WPFooter",
	"complementary": "WPSideBar",
	"table":         "Table",
	"grid":          "Table",
}

// section is a page section found in the HTML.
type section struct {
	PageElement
	n *html.Node
}

// skipped reports whether a visitor never sees the element or its contents:
// scripts, styles, noscript, template contents and hidden elements.
func skipped(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Noscript, atom.Template:
		return true
	}
	_, hidden := attr(n, "hidden")
	aria, _ := attr(n, "aria-hidden")
	return hidden || aria == "true"
}

// attr is an attribute's value and whether the element has it.
func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// visit calls fn on every element a visitor can see, in document order.
func visit(n *html.Node, fn func(*html.Node)) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if skipped(c) {
			continue
		}
		if c.Type == html.ElementNode {
			fn(c)
		}
		visit(c, fn)
	}
}

// htmlText is the visible text under n: every text node outside the skipped
// elements, whitespace collapsed.
func htmlText(doc *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if skipped(c) {
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

// sectionType is the type the element's section needs, or "". A header or
// footer inside an article, aside, main, nav or section belongs to that part,
// not the page, so it is not the page's WPHeader or WPFooter.
func sectionType(n *html.Node) string {
	role, _ := attr(n, "role")
	if role == "presentation" || role == "none" {
		return ""
	}
	if isAd(n) {
		return "WPAdBlock"
	}
	switch n.DataAtom {
	case atom.Nav:
		if isBreadcrumb(n) {
			return "BreadcrumbList"
		}
		return "SiteNavigationElement"
	case atom.Header, atom.Footer:
		if scoped(n) {
			return ""
		}
		if n.DataAtom == atom.Header {
			return "WPHeader"
		}
		return "WPFooter"
	case atom.Aside:
		return "WPSideBar"
	case atom.Table:
		return "Table"
	}
	if t := roleTypes[role]; t != "" {
		if t == "SiteNavigationElement" && isBreadcrumb(n) {
			return "BreadcrumbList"
		}
		return t
	}
	return ""
}

func isBreadcrumb(n *html.Node) bool {
	for _, k := range []string{"aria-label", "id", "class"} {
		if v, _ := attr(n, k); strings.Contains(strings.ToLower(v), "breadcrumb") {
			return true
		}
	}
	return false
}

func scoped(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		switch p.DataAtom {
		case atom.Article, atom.Aside, atom.Main, atom.Nav, atom.Section:
			return true
		}
	}
	return false
}

// isAd reports an ad slot: an AdSense ins, an element with an ad slot or
// client attribute, or an ad network iframe.
func isAd(n *html.Node) bool {
	class, _ := attr(n, "class")
	if n.DataAtom == atom.Ins && slices.Contains(strings.Fields(class), "adsbygoogle") {
		return true
	}
	for _, k := range []string{"data-ad-slot", "data-ad-client"} {
		if _, ok := attr(n, k); ok {
			return true
		}
	}
	if n.DataAtom == atom.Iframe {
		src, _ := attr(n, "src")
		return strings.Contains(src, "googlesyndication.com") || strings.Contains(src, "doubleclick.net")
	}
	return false
}

// sectionName is the section's aria-label, a table's caption, or else its
// first heading.
func sectionName(n *html.Node) string {
	if l, _ := attr(n, "aria-label"); strings.TrimSpace(l) != "" {
		return strings.TrimSpace(l)
	}
	var name string
	visit(n, func(c *html.Node) {
		if name != "" {
			return
		}
		switch c.DataAtom {
		case atom.Caption:
			if n.DataAtom == atom.Table {
				name = htmlText(c)
			}
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			name = htmlText(c)
		}
	})
	return name
}

var cssIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// selectorFor is a CSS selector that finds n alone on the page: its id, its
// tag and aria-label when no other element shares them, or else its path of
// child steps from the nearest ancestor with an id, or from body.
func selectorFor(doc, n *html.Node) string {
	if id, _ := attr(n, "id"); cssIdent.MatchString(id) {
		return "#" + id
	}
	if l, _ := attr(n, "aria-label"); l != "" {
		s := fmt.Sprintf("%s[aria-label=%q]", n.Data, l)
		if g, err := cascadia.ParseGroup(s); err == nil && len(cascadia.QueryAll(doc, g)) == 1 {
			return s
		}
	}
	var steps []string
	for e := n; e != nil && e.Type == html.ElementNode; e = e.Parent {
		if id, _ := attr(e, "id"); e != n && cssIdent.MatchString(id) {
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

// pageSections are the page's sections in document order.
func pageSections(doc *html.Node) []*section {
	var out []*section
	visit(doc, func(n *html.Node) {
		if t := sectionType(n); t != "" {
			out = append(out, &section{PageElement: PageElement{Type: t, Selector: selectorFor(doc, n), Name: sectionName(n)}, n: n})
		}
	})
	return out
}

// pageElements lists the page's sections and which draft node marks each up.
// An element node marks the sections of its type that its cssSelector
// selects; one without a cssSelector marks its type's only section. Any
// BreadcrumbList marks the breadcrumb navs. It warns on every section left
// unmarked and on a cssSelector that is not valid CSS, selects nothing, or
// selects only sections of another type.
func (c *checker) pageElements(all []located, doc *html.Node) []PageElement {
	sections := pageSections(doc)
	byType := map[string][]*section{}
	for _, s := range sections {
		byType[s.Type] = append(byType[s.Type], s)
	}
	mark := func(s *section, path string) {
		if s.MarkedBy == "" {
			s.MarkedBy = path
		}
	}
	bare := map[string][]located{} // element nodes without a cssSelector, by type
	for _, l := range all {
		types := typeNames(l.n)
		i := slices.IndexFunc(elementTypeNames, func(t string) bool { return slices.Contains(types, t) })
		if i < 0 {
			continue
		}
		t := elementTypeNames[i]
		if t == "BreadcrumbList" {
			for _, s := range byType[t] {
				mark(s, l.path)
			}
			continue
		}
		sels := strs(l.n, "cssSelector")
		if len(sels) == 0 {
			bare[t] = append(bare[t], l)
			continue
		}
		for _, sel := range sels {
			g, err := cascadia.ParseGroup(sel)
			if err != nil {
				c.add(l.path, Warning, "cssSelector %q is not a valid CSS selector", sel)
				continue
			}
			var other []string
			hit := false
			for _, s := range sections {
				if g.Match(s.n) {
					if s.Type == t {
						hit = true
						mark(s, l.path)
					} else {
						other = append(other, s.Type)
					}
				}
			}
			switch {
			case hit:
			case len(other) > 0:
				c.add(l.path, Warning, "cssSelector %q selects a %s, not a %s", sel, other[0], t)
			case len(cascadia.QueryAll(doc, g)) == 0:
				c.add(l.path, Warning, "cssSelector %q matches nothing on the page", sel)
			}
		}
	}
	for t, nodes := range bare {
		if len(nodes) == 1 && len(byType[t]) == 1 {
			mark(byType[t][0], nodes[0].path)
			continue
		}
		for _, l := range nodes {
			c.add(l.path, Warning, "%s has no cssSelector and the page has %d of its sections: give it the cssSelector of the one it marks up", t, len(byType[t]))
		}
	}
	out := make([]PageElement, len(sections))
	for i, s := range sections {
		out[i] = s.PageElement
		if s.MarkedBy != "" {
			continue
		}
		if s.Type == "BreadcrumbList" {
			c.add("html:"+s.Selector, Warning, "the page's breadcrumb <%s> has no BreadcrumbList: add one listing its links", s.n.Data)
			continue
		}
		c.add("html:"+s.Selector, Warning, `the page's <%s> has no %s: add {"@type": %q, "cssSelector": %q} to the WebPage's hasPart`, s.n.Data, s.Type, s.Type, s.Selector)
	}
	return out
}
