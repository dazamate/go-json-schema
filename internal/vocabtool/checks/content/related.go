package content

import (
	"net/url"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// relatedKind is what a related-reading block holds that the draft lacks.
type relatedKind int

const (
	relatedProducts relatedKind = iota // product cards the Product does not name
	relatedGuide                       // a guide's own pages no ItemList lists
	relatedLinks                       // pages of the site the WebPage's relatedLink does not list
)

// relatedFind is one block the page labels as related reading or products,
// and what it holds that the draft lacks.
type relatedFind struct {
	kind  relatedKind
	path  string
	label string   // the block's label
	names []string // relatedProducts: the products' names
	count int      // relatedGuide: how many pages
	links []string // relatedLinks: the addresses missing
}

// relatedKey is the memo key of the page's related blocks.
type relatedKey struct{}

// relatedOf compares the page's blocks of related reading with the draft:
// the products a block lists, the pages of a guide, and the links to other
// pages of the site. The three related checks share it. It needs the page's
// document.
func relatedOf(c *audit.Context) []relatedFind {
	return audit.Memo(c, relatedKey{}, func() []relatedFind {
		p, d := c.Page, c.Index()
		if p.Doc == nil {
			return nil
		}
		var rel []string
		for _, l := range c.Nodes {
			if c.Is(l.N, "WebPage") {
				for _, s := range jsonld.Strs(l.N, "relatedLink") {
					rel = append(rel, p.Abs(s))
				}
			}
		}
		var out []relatedFind
		for _, b := range p.RelatedBoxes() {
			t, box := b.Label, b.N
			path := "html:" + webpage.SelectorFor(p.Doc, box)
			var cards []*webpage.Card
			for _, cd := range p.ProductCards() {
				if cd.Related && slices.Contains(webpage.Ancestors(cd.N), box) {
					cards = append(cards, cd)
				}
			}
			if len(cards) > 0 {
				var names []string
				for _, cd := range cards {
					if !d.URLs[cd.URL] && !slices.Contains(names, cd.Name) {
						names = append(names, cd.Name)
					}
				}
				if len(names) > 0 {
					out = append(out, relatedFind{kind: relatedProducts, path: path, label: t, names: names})
				}
				continue
			}
			// a guide's own pages, listed on its hub, are a part of the page
			if webpage.GuideLabel.MatchString(t) {
				var pages []string
				for _, u := range p.SiteLinks(box) {
					if !slices.Contains(pages, u) {
						pages = append(pages, u)
					}
				}
				if len(pages) >= 3 && slices.ContainsFunc(pages, func(u string) bool { return !d.URLs[u] }) {
					out = append(out, relatedFind{kind: relatedGuide, path: path, label: t, count: len(pages)})
				}
				continue
			}
			var missing []string
			webpage.Visit(box, func(a *xhtml.Node) {
				if a.DataAtom != atom.A {
					return
				}
				h, _ := webpage.Attr(a, "href")
				pu, err := url.Parse(strings.TrimSpace(h))
				if h == "" || strings.HasPrefix(h, "#") || err != nil {
					return
				}
				if p.Base != nil {
					pu = p.Base.ResolveReference(pu)
				}
				if (p.Base != nil && !strings.EqualFold(pu.Hostname(), p.Base.Hostname())) || (p.Base == nil && pu.IsAbs()) {
					return
				}
				if link := pu.String(); !slices.Contains(rel, txt.NormURL(pu)) && !slices.Contains(missing, link) {
					missing = append(missing, link)
				}
			})
			if len(missing) > 0 {
				out = append(out, relatedFind{kind: relatedLinks, path: path, label: t, links: missing})
			}
		}
		return out
	})
}

// relatedOfKind are the related blocks' finds of the kind, on a page search
// engines index.
func relatedOfKind(c *audit.Context, kind relatedKind) []relatedFind {
	if !c.Indexed() {
		return nil
	}
	var out []relatedFind
	for _, f := range relatedOf(c) {
		if f.kind == kind {
			out = append(out, f)
		}
	}
	return out
}
