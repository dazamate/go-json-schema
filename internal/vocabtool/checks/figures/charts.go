package figures

import (
	"fmt"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

func init() { audit.Register(Charts{}) }

// Charts warns on figures drawn in SVG, with a caption, that no ImageObject
// carries: the images check reads only img elements. It needs the page's
// document, on a page search engines index.
type Charts struct{}

// ID is the check's ID.
func (Charts) ID() string { return "figures/charts" }

// Run reports the page's captioned charts in one issue, at the first.
func (Charts) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p, d := c.Page, c.Index()
	var first *xhtml.Node
	var caps []string
	for _, f := range webpage.QueryAll(p.Doc, "figure") {
		fc := webpage.QueryAll(f, "figcaption")
		if webpage.Chrome(f) || !webpage.Shown(f) || len(fc) == 0 || len(webpage.QueryAll(f, "svg")) == 0 || len(webpage.QueryAll(f, "img, video, iframe, table")) > 0 {
			continue
		}
		cp := webpage.HTMLText(fc[0])
		if cp == "" || d.HoldsText(cp) {
			continue
		}
		if first == nil {
			first = f
		}
		caps = append(caps, fmt.Sprintf("%q", txt.Clip(cp, 80)))
	}
	if first == nil {
		return nil
	}
	what := "the chart " + caps[0]
	if len(caps) > 1 {
		what = fmt.Sprintf("%d charts (%s)", len(caps), txt.Examples(caps))
	}
	return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, first), audit.Warning, "the page draws %s with no ImageObject: add one for each (caption word for word, encodingFormat image/svg+xml, about the Dataset it draws) as the image of the WebPage or the Dataset", what)}
}
