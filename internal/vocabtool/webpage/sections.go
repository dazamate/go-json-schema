package webpage

import (
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// roleTypes are the ARIA landmark and table roles that name a section type.
var roleTypes = map[string]string{
	"navigation":    "SiteNavigationElement",
	"banner":        "WPHeader",
	"contentinfo":   "WPFooter",
	"complementary": "WPSideBar",
	"table":         "Table",
	"grid":          "Table",
}

// Section is a page section found in the HTML: the type it needs, a CSS
// selector that finds it alone, its name and its element.
type Section struct {
	Type     string
	Selector string
	Name     string
	N        *html.Node
}

// sectionType is the type the element's section needs, or "". A header or
// footer inside an article, aside, main, nav, section, header or footer
// belongs to that part, not the page, so it is not the page's WPHeader or
// WPFooter; nor is one in a drawer or dialog. A block its inline style hides
// is no section, and a nav inside a nav is a part of it.
func sectionType(n *html.Node) string {
	role, _ := Attr(n, "role")
	if role == "presentation" || role == "none" || StyleHidden(n) {
		return ""
	}
	if isAd(n) {
		return "WPAdBlock"
	}
	t := ""
	switch n.DataAtom {
	case atom.Nav:
		t = "SiteNavigationElement"
	case atom.Header:
		t = "WPHeader"
	case atom.Footer:
		t = "WPFooter"
	case atom.Aside:
		t = "WPSideBar"
		// an aside the page itself calls its banner is its header
		if r := roleTypes[role]; r != "" {
			t = r
		}
	case atom.Table:
		if !dataTable(n) {
			return ""
		}
		return "Table"
	case atom.Ol, atom.Ul, atom.Div, atom.P:
		if trailBlock(n) {
			return "BreadcrumbList"
		}
		t = roleTypes[role]
	default:
		t = roleTypes[role]
	}
	switch t {
	case "SiteNavigationElement":
		if isBreadcrumb(n) {
			return "BreadcrumbList"
		}
		if inNav(n) {
			return ""
		}
	case "WPHeader", "WPFooter":
		if (n.DataAtom == atom.Header || n.DataAtom == atom.Footer) && scoped(n) || Overlay(n) {
			return ""
		}
	case "WPSideBar":
		if inTool(n) || summaryBlock(n) || Overlay(n) || consentNotice(n) {
			return ""
		}
	}
	return t
}

// crumbPart is a word of a class or id that makes a breadcrumb name a part
// of the trail or a block beside it ("breadcrumb-item",
// "content-after-breadcrumb"), not the trail.
var crumbPart = regexp.MustCompile(`^(?:item|items|link|links|sep|separator|current|label|icon|arrow|after|before|skip|content|text|home|last)$`)

// namesTrail reports a class or id that names a breadcrumb trail itself:
// one of its classes, or its id, holds breadcrumb or breadcrumbs among words
// that name no part of it.
func namesTrail(n *html.Node) bool {
	id, _ := Attr(n, "id")
	class, _ := Attr(n, "class")
	for _, s := range append(strings.Fields(class), id) {
		words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return r == '-' || r == '_' })
		if slices.ContainsFunc(words, func(w string) bool { return w == "breadcrumb" || w == "breadcrumbs" }) &&
			!slices.ContainsFunc(words, crumbPart.MatchString) {
			return true
		}
	}
	return false
}

// trailBlock reports a breadcrumb trail outside a nav: a list or block
// named a breadcrumb that links two pages or more, outside any nav, and
// holding no nav or other trail, so a wrapper around a trail is not one.
func trailBlock(n *html.Node) bool {
	if role, _ := Attr(n, "role"); role != "" || !namesTrail(n) || len(QueryAll(n, "a[href]")) < 2 || inNav(n) {
		return false
	}
	for _, c := range QueryAll(n, "nav, [role=navigation], ol, ul, div, p") {
		if c.DataAtom == atom.Nav || namesTrail(c) {
			return false
		}
	}
	return true
}

// inNav reports an element inside a nav: a nav element or a navigation
// landmark.
func inNav(n *html.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if role, _ := Attr(e, "role"); e.DataAtom == atom.Nav || role == "navigation" {
			return true
		}
	}
	return false
}

// consentWord is a word of an id, class or label that names a cookie or
// consent notice.
var consentWord = regexp.MustCompile(`^(?:cookies?|consent|gdpr)$`)

// consentNotice reports a cookie or consent notice, by its id, class or
// label ("moove_gdpr_cookie_info_bar").
func consentNotice(n *html.Node) bool {
	for _, k := range []string{"id", "class", "aria-label", "aria-labelledby"} {
		v, _ := Attr(n, k)
		for _, w := range strings.FieldsFunc(strings.ToLower(v), func(r rune) bool { return !('a' <= r && r <= 'z') }) {
			if consentWord.MatchString(w) {
				return true
			}
		}
	}
	return false
}

// dataTable reports a table that holds data, not layout: one with a header
// cell, or a row of two cells with text. A row of a photo beside a quote, or
// of linked logos, lays the page out.
func dataTable(n *html.Node) bool {
	if len(QueryAll(n, "th")) > 0 {
		return true
	}
	for _, tr := range QueryAll(n, "tr") {
		cells := 0
		for c := tr.FirstChild; c != nil; c = c.NextSibling {
			if (c.DataAtom == atom.Td || c.DataAtom == atom.Th) && strings.TrimSpace(HTMLText(c)) != "" {
				cells++
			}
		}
		if cells >= 2 {
			return true
		}
	}
	return false
}

// isBreadcrumb reports a nav that is a breadcrumb trail, by its label, id or
// class.
func isBreadcrumb(n *html.Node) bool {
	for _, k := range []string{"aria-label", "id", "class"} {
		if v, _ := Attr(n, k); strings.Contains(strings.ToLower(v), "breadcrumb") {
			return true
		}
	}
	return false
}

// inTool reports an element inside a form or a tool: a block of the same
// section that holds two or more form controls, outside any form, besides
// the element's own. An aside beside a contact form is still a sidebar.
func inTool(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		switch p.DataAtom {
		case atom.Form:
			return true
		case atom.Main, atom.Body, atom.Article, atom.Section, atom.Html:
			return false
		}
		if controls(p)-controls(n) >= 2 {
			return true
		}
	}
	return false
}

// controls counts the visible form controls under n outside forms.
func controls(n *html.Node) int {
	k := 0
	Visit(n, func(c *html.Node) {
		for e := c.Parent; e != nil && e != n.Parent; e = e.Parent {
			if e.DataAtom == atom.Form {
				return
			}
		}
		switch c.DataAtom {
		case atom.Input:
			if t, _ := Attr(c, "type"); !strings.EqualFold(t, "hidden") {
				k++
			}
		case atom.Select, atom.Textarea:
			k++
		}
	})
	return k
}

// SummaryLabel matches the label of a block that sums the page up.
var SummaryLabel = regexp.MustCompile(`(?i)\bkey (?:takeaways?|points)\b|\btakeaways\b|\bsummary\b|\btl;?dr\b|\bat a glance\b|\bin brief\b|\bin short\b`)

// summaryBlock reports a block that sums the page up (Key takeaways, TL;DR,
// At a glance), by its label, class or heading.
func summaryBlock(n *html.Node) bool {
	label, _ := Attr(n, "aria-label")
	class, _ := Attr(n, "class")
	return SummaryLabel.MatchString(label) || SummaryLabel.MatchString(class) || SummaryLabel.MatchString(SectionName(n))
}

// isAd reports an ad slot: an AdSense ins, an element with an ad slot or
// client attribute, or an ad network iframe.
func isAd(n *html.Node) bool {
	class, _ := Attr(n, "class")
	if n.DataAtom == atom.Ins && slices.Contains(strings.Fields(class), "adsbygoogle") {
		return true
	}
	for _, k := range []string{"data-ad-slot", "data-ad-client"} {
		if _, ok := Attr(n, k); ok {
			return true
		}
	}
	if n.DataAtom == atom.Iframe {
		src, _ := Attr(n, "src")
		return strings.Contains(src, "googlesyndication.com") || strings.Contains(src, "doubleclick.net")
	}
	return false
}

// SectionName is the section's aria-label, a table's caption, or else its
// first heading.
func SectionName(n *html.Node) string {
	if l, _ := Attr(n, "aria-label"); strings.TrimSpace(l) != "" {
		return strings.TrimSpace(l)
	}
	var name string
	Visit(n, func(c *html.Node) {
		if name != "" {
			return
		}
		switch c.DataAtom {
		case atom.Caption:
			if n.DataAtom == atom.Table {
				name = HTMLText(c)
			}
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			name = HTMLText(c)
		}
	})
	return name
}

// PageSections are the page's sections in document order.
func PageSections(doc *html.Node) []*Section {
	var out []*Section
	Visit(doc, func(n *html.Node) {
		if t := sectionType(n); t != "" {
			out = append(out, &Section{Type: t, Selector: SelectorFor(doc, n), Name: SectionName(n), N: n})
		}
	})
	return out
}
