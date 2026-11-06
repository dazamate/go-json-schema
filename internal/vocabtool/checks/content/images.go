package content

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// imageFind is one content image and what the draft lacks for it.
type imageFind struct {
	path string   // the image's path, or its ImageObject's for an incomplete one
	alt  string   // the image's alt text
	miss []string // what the ImageObject lacks, for an incomplete one
}

// imaged is how the draft covers the page's content images.
type imaged struct {
	unnamed    []imageFind // an image no ImageObject or image URL names
	incomplete []imageFind // an ImageObject for an image that lacks its size or description
}

// imagesKey is the memo key of the page's images.
type imagesKey struct{}

// attrInt is the number an attribute holds, 0 for none.
func attrInt(n *xhtml.Node, key string) int {
	v, _ := webpage.Attr(n, key)
	i, _ := strconv.Atoi(strings.TrimSpace(v))
	return i
}

// imgURLs are an image's src and srcset URLs, resolved.
func imgURLs(p *webpage.Facts, n *xhtml.Node) []string {
	var out []string
	if s, _ := webpage.Attr(n, "src"); s != "" {
		out = append(out, p.Abs(s))
	}
	srcset, _ := webpage.Attr(n, "srcset")
	for _, part := range strings.Split(srcset, ",") {
		if f := strings.Fields(part); len(f) > 0 {
			out = append(out, p.Abs(f[0]))
		}
	}
	return out
}

// imagesOf compares the page's content images (an image with alt text,
// outside the page's header, footer and navs, and not a product card's) with
// the draft's ImageObjects and image URLs. The two image checks share it. It
// needs the page's document.
func imagesOf(c *audit.Context) imaged {
	return audit.Memo(c, imagesKey{}, func() imaged {
		p := c.Page
		if p.Doc == nil {
			return imaged{}
		}
		objs := map[string]jsonld.Node{}
		plain := map[string]bool{}
		for _, l := range c.Nodes {
			if c.Is(l.N, "ImageObject") {
				for _, k := range []string{"contentUrl", "url", "thumbnailUrl"} {
					for _, s := range jsonld.Strs(l.N, k) {
						objs[p.Abs(s)] = l
					}
				}
				continue
			}
			for _, k := range []string{"image", "logo", "photo", "primaryImageOfPage", "thumbnailUrl"} {
				for _, s := range jsonld.Strs(l.N, k) {
					plain[p.Abs(s)] = true
				}
			}
		}
		var out imaged
		reported := map[string]bool{}
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			// a product card's image is its Product's, which the listing check asks for
			if n.DataAtom != atom.Img || webpage.Chrome(n) || p.InCard(n) {
				return
			}
			alt, _ := webpage.Attr(n, "alt")
			w, h := attrInt(n, "width"), attrInt(n, "height")
			if strings.TrimSpace(alt) == "" || (w > 0 && w <= 2) || (h > 0 && h <= 2) {
				return
			}
			var obj *jsonld.Node
			covered := false
			for _, u := range imgURLs(p, n) {
				if o, ok := objs[u]; ok {
					obj, covered = &o, true
					break
				}
				covered = covered || plain[u]
			}
			if !covered {
				out.unnamed = append(out.unnamed, imageFind{path: "html:" + webpage.SelectorFor(p.Doc, n), alt: txt.Clip(alt, 60)})
				return
			}
			if obj == nil || reported[obj.Path] {
				return
			}
			var miss []string
			if w > 0 && len(jsonld.Scalars(obj.N, "width")) == 0 {
				miss = append(miss, fmt.Sprintf("width %d", w))
			}
			if h > 0 && len(jsonld.Scalars(obj.N, "height")) == 0 {
				miss = append(miss, fmt.Sprintf("height %d", h))
			}
			if len(jsonld.Strs(obj.N, "description"))+len(jsonld.Strs(obj.N, "caption")) == 0 {
				miss = append(miss, "a description from its alt text")
			}
			if len(miss) > 0 {
				reported[obj.Path] = true
				out.incomplete = append(out.incomplete, imageFind{path: obj.Path, alt: txt.Clip(alt, 60), miss: miss})
			}
		})
		return out
	})
}
