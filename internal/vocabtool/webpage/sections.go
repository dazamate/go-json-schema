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
// footer inside an article, aside, main, nav or section belongs to that part,
// not the page, so it is not the page's WPHeader or WPFooter.
func sectionType(n *html.Node) string {
	role, _ := Attr(n, "role")
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
		if inTool(n) || summaryBlock(n) {
			return ""
		}
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
