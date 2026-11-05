package content

import (
	"net/url"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(OutsideLinks{}) }

// OutsideLinks warns on a group of three or more links to other sites, under
// one ancestor with an id, whose URLs no node in the draft names: partners,
// vendors or sources the page features. The business's own profiles and share
// buttons are left to their checks. It needs the page's document and its
// address, on a page search engines index.
type OutsideLinks struct{}

// ID is the check's ID.
func (OutsideLinks) ID() string { return "content/outside-links" }

// Run reports each such group.
func (OutsideLinks) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || c.Page.Base == nil {
		return nil
	}
	p, d := c.Page, c.Index()
	type group struct{ urls, labels []string }
	var order []*xhtml.Node
	groups := map[*xhtml.Node]*group{}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.A {
			return
		}
		h, _ := webpage.Attr(n, "href")
		u, err := url.Parse(strings.TrimSpace(h))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || strings.EqualFold(u.Hostname(), p.Base.Hostname()) {
			return
		}
		// the business's profiles and share buttons have checks of their own,
		// and a feed of its posts features no outside business
		if webpage.SocialLink(h) || webpage.ShareLink.MatchString(h) {
			return
		}
		key := txt.NormURL(u)
		if d.URLs[key] {
			return
		}
		box := webpage.IDBox(n)
		g := groups[box]
		if g == nil {
			g = &group{}
			groups[box] = g
			order = append(order, box)
		}
		if !slices.Contains(g.urls, key) {
			g.urls = append(g.urls, key)
			label := strings.TrimSpace(webpage.HTMLText(n))
			if label == "" {
				label, _ = webpage.Attr(n, "aria-label")
			}
			if label = txt.Clip(label, 40); label != "" && !slices.Contains(g.labels, label) {
				g.labels = append(g.labels, label)
			}
		}
	})
	var out []audit.Issue
	for _, box := range order {
		g := groups[box]
		if len(g.urls) < 3 {
			continue
		}
		labels := g.labels
		if len(labels) > 8 {
			labels = append(labels[:8:8], "…")
		}
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, box), audit.Warning, "%d links to outside sites here (%s) name no node: when they are businesses or people the page features, add an ItemList of them to the WebPage's hasPart, each with url and sameAs; when they are sources, list them as citation", len(g.urls), strings.Join(labels, ", ")))
	}
	return out
}
