package audit

import (
	"net/url"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// MainWebPage is the draft's WebPage for the page: a node of the graph's top
// level before one nested in another (a catalog's link to the same page),
// the one whose url is the page's before the first that is not an FAQPage
// part. It reports false when the draft has no WebPage that is more than a
// reference.
func MainWebPage(v *vocab.Vocab, nodes []jsonld.Node, p *webpage.Facts) (jsonld.Node, bool) {
	var best jsonld.Node
	rank := 0
	for _, l := range nodes {
		if !v.Is(jsonld.TypeNames(l.N), "WebPage") || len(l.N.Keys) < 2 {
			continue
		}
		r := 0
		if !v.Is(jsonld.TypeNames(l.N), "FAQPage") {
			r = 1
		}
		if p.URL != nil {
			for _, s := range append(jsonld.Strs(l.N, "url"), jsonld.Strs(l.N, "@id")...) {
				if u, err := url.Parse(s); err == nil && u.Fragment == "" && txt.NormURL(u) == txt.NormURL(p.URL) {
					r = 2
				}
			}
		}
		if r > 0 && jsonld.TopLevel(l.Path) {
			r += 2
		}
		if r > rank {
			best, rank = l, r
		}
	}
	return best, rank > 0
}
