package onpage

import (
	"slices"

	"github.com/andybalholm/cascadia"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
)

// PageElement is a section of the page that a specific schema.org type
// describes: a WebPageElement subtype, or a BreadcrumbList for a breadcrumb
// nav. MarkedBy is the path of the draft node that marks it up, "" for none.
type PageElement struct {
	Type     string `json:"type"`
	Selector string `json:"selector"`
	Name     string `json:"name,omitempty"`
	MarkedBy string `json:"marked_by,omitempty"`
}

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

// analysis is what the draft and the page's sections say of each other: the
// page elements, and the section checks' findings, one list for each.
type analysis struct {
	elements []PageElement

	invalid    []selectorFinding // a cssSelector that is not valid CSS
	wrongType  []selectorFinding // a cssSelector that selects only sections of another type
	nothing    []selectorFinding // a cssSelector that selects nothing
	bare       []bareFinding     // an element node without a cssSelector, where it marks no one section
	breadcrumb []sectionFinding  // a breadcrumb nav no BreadcrumbList marks
	unmarked   []sectionFinding  // any other section no element node marks
}

// The findings are raw: each check words its own message from them.

// selectorFinding is an element node's cssSelector that went wrong: the
// node's path, the selector, the type of section it selects (other than the
// node's) and the type the node is.
type selectorFinding struct {
	path, selector, selected, want string
}

// bareFinding is an element node without a cssSelector that marks no one
// section: the node's path, its type and how many sections of that type the
// page has.
type bareFinding struct {
	path, ty string
	sections int
}

// sectionFinding is a page section no node marks up: its selector, its tag
// and its type.
type sectionFinding struct {
	selector, tag, ty string
}

// analysisKey is the memo key of the analysis.
type analysisKey struct{}

// Elements are the page's sections and which draft node marks each up, in
// document order; nil without page_html. An element node marks the sections
// of its type that its cssSelector selects; one without a cssSelector marks
// its type's only section. Any BreadcrumbList marks the breadcrumb navs. A
// page search engines skip lists its sections, unmarked: nothing is asked of
// its markup. The slice is shared: read it only.
func Elements(c *audit.Context) []PageElement { return analyse(c).elements }

// analyse reads the page's sections against the draft, once. It finds the
// section checks' findings on the way, because the nodes that mark the sections
// are the ones those findings are about.
func analyse(c *audit.Context) *analysis {
	return audit.Memo(c, analysisKey{}, func() *analysis {
		a := &analysis{}
		if c.HTML == nil {
			return a
		}
		secs := sections(c.HTML)
		if c.Page != nil && c.Page.Private != "" {
			// a page search engines skip: list its sections, ask for nothing
			a.elements = make([]PageElement, len(secs))
			for i, s := range secs {
				a.elements[i] = s.PageElement
			}
			return a
		}
		a.mark(c, secs)
		return a
	})
}

// mark marks the sections up from the draft's element nodes, finding the
// section checks' issues, and lists the sections.
func (a *analysis) mark(c *audit.Context, secs []*section) {
	byType := map[string][]*section{}
	for _, s := range secs {
		byType[s.Type] = append(byType[s.Type], s)
	}
	mark := func(s *section, path string) {
		if s.MarkedBy == "" {
			s.MarkedBy = path
		}
	}
	bare := map[string][]jsonld.Node{} // element nodes without a cssSelector, by type
	for _, l := range c.Nodes {
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
				a.invalid = append(a.invalid, selectorFinding{path: l.Path, selector: sel})
				continue
			}
			var other []string
			hit := false
			for _, s := range secs {
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
				a.wrongType = append(a.wrongType, selectorFinding{path: l.Path, selector: sel, selected: other[0], want: t})
			case len(cascadia.QueryAll(c.HTML, g)) == 0:
				a.nothing = append(a.nothing, selectorFinding{path: l.Path, selector: sel})
			}
		}
	}
	for _, t := range elementTypeNames {
		nodes := bare[t]
		if len(nodes) == 1 && len(byType[t]) == 1 {
			mark(byType[t][0], nodes[0].Path)
			continue
		}
		for _, l := range nodes {
			a.bare = append(a.bare, bareFinding{path: l.Path, ty: t, sections: len(byType[t])})
		}
	}
	a.elements = make([]PageElement, len(secs))
	for i, s := range secs {
		a.elements[i] = s.PageElement
		switch {
		case s.MarkedBy != "":
		case s.Type == "BreadcrumbList":
			a.breadcrumb = append(a.breadcrumb, sectionFinding{selector: s.Selector, tag: s.n.Data, ty: s.Type})
		default:
			a.unmarked = append(a.unmarked, sectionFinding{selector: s.Selector, tag: s.n.Data, ty: s.Type})
		}
	}
}
