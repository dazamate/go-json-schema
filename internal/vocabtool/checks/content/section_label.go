package content

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(SectionLabel{}) }

// SectionLabel warns when a short label sits right above the page's h1 (a
// kicker naming the section) and the Article does not carry it as its
// articleSection. It needs the page's document, on a page search engines
// index.
type SectionLabel struct{}

// ID is the check's ID.
func (SectionLabel) ID() string { return "content/section-label" }

// Run reports the Article that lacks the label as its articleSection.
func (SectionLabel) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p := c.Page
	i := slices.IndexFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "Article") })
	if i < 0 {
		return nil
	}
	var h1 *xhtml.Node
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if h1 == nil && n.DataAtom == atom.H1 {
			h1 = n
		}
	})
	if h1 == nil {
		return nil
	}
	prev := h1.PrevSibling
	for prev != nil && prev.Type != xhtml.ElementNode {
		prev = prev.PrevSibling
	}
	if prev == nil || !slices.Contains([]atom.Atom{atom.P, atom.Span, atom.Div, atom.A}, prev.DataAtom) || len(webpage.QueryAll(prev, "time, nav, a[href]")) > 0 {
		return nil
	}
	label := webpage.HTMLText(prev)
	if n := len(strings.Fields(label)); n == 0 || n > 5 {
		return nil
	}
	art := c.Nodes[i]
	if slices.ContainsFunc(jsonld.Strs(art.N, "articleSection"), func(s string) bool { return txt.NormWords(s) == txt.NormWords(label) }) {
		return nil
	}
	return []audit.Issue{audit.Issuef(art.Path, audit.Warning, "the page labels the article %q above its h1: set articleSection %q", label, label)}
}
