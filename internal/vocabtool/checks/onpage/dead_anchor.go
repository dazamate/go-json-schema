package onpage

import (
	"net/url"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
)

func init() { audit.Register(DeadAnchor{}) }

// DeadAnchor warns about a link to this page whose #fragment is no id on it:
// a step's url that points at a section the page does not have. It needs the
// page's HTML and its address, and holds a page search engines skip to it too.
type DeadAnchor struct{}

// ID is the check's ID.
func (DeadAnchor) ID() string { return "onpage/dead-anchor" }

// Run reports each link to this page whose fragment names no element.
func (DeadAnchor) Run(c *audit.Context) []audit.Issue {
	p := c.Page
	if p == nil || p.Doc == nil || p.Base == nil {
		return nil
	}
	ids := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if id, ok := webpage.Attr(n, "id"); ok && n.Type == html.ElementNode {
			ids[id] = true
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(p.Doc)
	page := txt.NormURL(p.Base)
	var out []audit.Issue
	for _, l := range c.Nodes {
		for _, k := range l.N.Keys {
			if strings.HasPrefix(k, "@") || k == "sameAs" {
				continue
			}
			for _, v := range jsonld.Values(l.N.Vals[k]) {
				s, _ := v.(string)
				u, err := url.Parse(s)
				if s == "" || err != nil || u.Fragment == "" || !u.IsAbs() || txt.NormURL(u) != page {
					continue
				}
				if !ids[u.Fragment] {
					out = append(out, audit.Issuef(l.Path+"."+k, audit.Warning, "%s points at #%s, which is no id on the page", s, u.Fragment))
				}
			}
		}
	}
	return out
}
