package vocabtool

import (
	"slices"

	"github.com/andybalholm/cascadia"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
)

// elementTypeNames are the types a page section can need: the WebPageElement
// subtypes, and BreadcrumbList for a breadcrumb nav.
var elementTypeNames = []string{"SiteNavigationElement", "WPHeader", "WPFooter", "WPSideBar", "Table", "WPAdBlock", "BreadcrumbList"}

// section is a page section found in the HTML, with the draft node that
// marks it up.
type section struct {
	PageElement
	n *html.Node
}

// sections are the page's sections in document order.
func sections(doc *html.Node) []*section {
	var out []*section
	for _, s := range webpage.PageSections(doc) {
		out = append(out, &section{PageElement: PageElement{Type: s.Type, Selector: s.Selector, Name: s.Name}, n: s.N})
	}
	return out
}

// pageElements lists the page's sections and which draft node marks each up.
// An element node marks the sections of its type that its cssSelector
// selects; one without a cssSelector marks its type's only section. Any
// BreadcrumbList marks the breadcrumb navs. It warns on every section left
// unmarked and on a cssSelector that is not valid CSS, selects nothing, or
// selects only sections of another type.
func (c *checker) pageElements(all []jsonld.Node, doc *html.Node, warn bool) []PageElement {
	if !warn {
		// a page search engines skip: list its sections, ask for nothing
		sections := sections(doc)
		out := make([]PageElement, len(sections))
		for i, s := range sections {
			out[i] = s.PageElement
		}
		return out
	}
	sections := sections(doc)
	byType := map[string][]*section{}
	for _, s := range sections {
		byType[s.Type] = append(byType[s.Type], s)
	}
	mark := func(s *section, path string) {
		if s.MarkedBy == "" {
			s.MarkedBy = path
		}
	}
	bare := map[string][]jsonld.Node{} // element nodes without a cssSelector, by type
	for _, l := range all {
		types := jsonld.TypeNames(l.N)
		i := slices.IndexFunc(elementTypeNames, func(t string) bool { return slices.Contains(types, t) })
		if i < 0 {
			continue
		}
		t := elementTypeNames[i]
		if t == "BreadcrumbList" {
			for _, s := range byType[t] {
				mark(s, l.Path)
			}
			continue
		}
		sels := jsonld.Strs(l.N, "cssSelector")
		if len(sels) == 0 {
			bare[t] = append(bare[t], l)
			continue
		}
		for _, sel := range sels {
			g, err := cascadia.ParseGroup(sel)
			if err != nil {
				c.add(l.Path, Warning, "cssSelector %q is not a valid CSS selector", sel)
				continue
			}
			var other []string
			hit := false
			for _, s := range sections {
				if g.Match(s.n) {
					if s.Type == t {
						hit = true
						mark(s, l.Path)
					} else {
						other = append(other, s.Type)
					}
				}
			}
			switch {
			case hit:
			case len(other) > 0:
				c.add(l.Path, Warning, "cssSelector %q selects a %s, not a %s", sel, other[0], t)
			case len(cascadia.QueryAll(doc, g)) == 0:
				c.add(l.Path, Warning, "cssSelector %q matches nothing on the page", sel)
			}
		}
	}
	for t, nodes := range bare {
		if len(nodes) == 1 && len(byType[t]) == 1 {
			mark(byType[t][0], nodes[0].Path)
			continue
		}
		for _, l := range nodes {
			c.add(l.Path, Warning, "%s has no cssSelector and the page has %d of its sections: give it the cssSelector of the one it marks up", t, len(byType[t]))
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
