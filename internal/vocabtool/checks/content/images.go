package content

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

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

// lazySrc are the attributes lazy-loading scripts keep an image's real
// address in, while src holds a placeholder.
var lazySrc = []string{"data-src", "data-lazy-src", "data-original", "data-lazy"}

// lazySrcset are the same for the srcset.
var lazySrcset = []string{"data-srcset", "data-lazy-srcset"}

// sizeSuffix is the size a CMS or CDN puts in an image's file name for a
// resized copy: WordPress "-300x200", Shopify "_533x" or "_533x533".
var sizeSuffix = regexp.MustCompile(`(?i)([-_]\d{2,4}x\d{0,4}|-scaled)(\.[a-z0-9]{3,4})?$`)

// imageKey is what two addresses of the same picture share: the host and
// path without the scheme, query and size suffix. The query stays for an
// address with no file extension.
func imageKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	path := u.Path
	ext := ""
	if i := strings.LastIndex(path, "."); i > strings.LastIndex(path, "/") && i > 0 {
		path, ext = path[:i], path[i:]
	}
	path = sizeSuffix.ReplaceAllString(path+ext, "$2")
	// an address with no file name (an image service) says which picture in
	// its query
	if ext == "" {
		path += "?" + u.RawQuery
	}
	return strings.ToLower(u.Host) + path
}

// notContent is the alt text of a picture that is not the page's content: a
// logo (the site's own, or a partner's), an icon or arrow, a close or consent
// button, a link's "opens in a new window" mark, a stand-in for a missing
// image, a "powered by" badge.
var notContent = regexp.MustCompile(`(?i)^(close|menu|search|loading|image|photo|picture|spacer|pixel)$|\b(icons?|arrows?|chevrons?|carets?|spinner|navicon|opens? in( a)? new (window|tab)|no image( available)?|image not available|thumbnail)\b|\b(button|loading|placeholder)$|^powered by\b|^\d(\.\d)? ?(out of 5 )?stars?$`)

// controlAlt is the alt text of a control the page draws with a picture: a
// gallery's previous or next, a wishlist or share heart, a zoom. Only a short
// alt text is one: a product may carry the word.
var controlAlt = regexp.MustCompile(`(?i)\b(wishlist|wish list|share|zoom|previous|next|prev|expand|collapse|toggle|play|pause|mute)\b`)

// urlAlt is an alt text that holds an address: a tracking pixel's.
var urlAlt = regexp.MustCompile(`(?i)\bhttps?[-:/]`)

// iconFile is a file name or folder that says the picture is an icon.
var iconFile = regexp.MustCompile(`(?i)(^|[/_.-])(icons?|noun|sprites?|emoji|favicons?)([/_.\d-]|$)`)

// logoAlt is an alt text that names a logo.
var logoAlt = regexp.MustCompile(`(?i)\blogos?\b`)

// logoOnA is an alt text of a picture of something that wears a logo.
var logoOnA = regexp.MustCompile(`(?i)\b(with|featuring|bearing|embroidered|printed|showing|wearing)\b[^.]*\blogos?\b`)

// decorative reports an alt text that names no content: nothing to read in
// it (an emoji), or one of notContent.
func decorative(alt string) bool {
	alt = strings.TrimSpace(alt)
	if urlAlt.MatchString(alt) {
		return true
	}
	alt = strings.NewReplacer("_", " ", "-", " ").Replace(alt)
	if strings.IndexFunc(alt, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0 {
		return true
	}
	if len(strings.Fields(alt)) <= 4 && controlAlt.MatchString(alt) {
		return true
	}
	return notContent.MatchString(alt) || (logoAlt.MatchString(alt) && !logoOnA.MatchString(alt))
}

// iconSrc reports an image all of whose files are icons.
func iconSrc(urls []string) bool {
	for _, u := range urls {
		if !iconFile.MatchString(u[strings.Index(u, "//")+2:]) {
			return false
		}
	}
	return len(urls) > 0
}

// inButton reports an image in a button, which is a control's face.
func inButton(n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.Button {
			return true
		}
	}
	return false
}

// linksElsewhere reports an image inside a link to another page: a teaser's
// thumbnail or a category tile, which is that page's picture. A link to the
// page itself, to the top of it or to a bigger copy of the picture does not
// count.
func linksElsewhere(p *webpage.Facts, n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom != atom.A {
			continue
		}
		href, _ := webpage.Attr(e, "href")
		href = strings.TrimSpace(href)
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return false
		}
		u, err := url.Parse(p.Abs(href))
		if err != nil || imageFile.MatchString(u.Path) {
			return false
		}
		return p.Base == nil || txt.NormURL(&url.URL{Scheme: u.Scheme, Host: u.Host, Path: u.Path}) != txt.NormURL(&url.URL{Scheme: p.Base.Scheme, Host: p.Base.Host, Path: p.Base.Path})
	}
	return false
}

// imageFile is the path of an image file.
var imageFile = regexp.MustCompile(`(?i)\.(jpe?g|png|webp|gif|avif|svg)$`)

// imgURLs are an image's src and srcset URLs, resolved: the ones a lazy
// loader keeps in data attributes and the ones of a picture's sources too.
// A data URI is a placeholder, not an address.
func imgURLs(p *webpage.Facts, n *xhtml.Node) []string {
	var out []string
	add := func(raw string) {
		if raw = strings.TrimSpace(raw); raw != "" && !strings.HasPrefix(strings.ToLower(raw), "data:") {
			out = append(out, p.Abs(raw))
		}
	}
	addSet := func(set string) {
		for _, part := range strings.Split(set, ",") {
			if f := strings.Fields(part); len(f) > 0 {
				add(f[0])
			}
		}
	}
	s, _ := webpage.Attr(n, "src")
	add(s)
	for _, k := range lazySrc {
		s, _ := webpage.Attr(n, k)
		add(s)
	}
	for _, k := range append([]string{"srcset"}, lazySrcset...) {
		s, _ := webpage.Attr(n, k)
		addSet(s)
	}
	if n.Parent != nil && n.Parent.DataAtom == atom.Picture {
		for c := n.Parent.FirstChild; c != nil; c = c.NextSibling {
			if c.DataAtom == atom.Source {
				for _, k := range append([]string{"srcset"}, lazySrcset...) {
					s, _ := webpage.Attr(c, k)
					addSet(s)
				}
			}
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
						objs[imageKey(p.Abs(s))] = l
					}
				}
				continue
			}
			for _, k := range []string{"image", "logo", "photo", "primaryImageOfPage", "thumbnailUrl"} {
				for _, s := range jsonld.Strs(l.N, k) {
					plain[imageKey(p.Abs(s))] = true
				}
			}
		}
		var out imaged
		reported := map[string]bool{}
		seen := map[string]bool{}
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			// a product card's image is its Product's, which the listing check asks for
			if n.DataAtom != atom.Img || webpage.Chrome(n) || p.InCard(n) || linksElsewhere(p, n) {
				return
			}
			alt, _ := webpage.Attr(n, "alt")
			w, h := attrInt(n, "width"), attrInt(n, "height")
			urls := imgURLs(p, n)
			if inButton(n) || strings.TrimSpace(alt) == "" || (w > 0 && w <= 2) || (h > 0 && h <= 2) || len(urls) == 0 || decorative(alt) || iconSrc(urls) {
				return
			}
			var obj *jsonld.Node
			covered := false
			for _, u := range urls {
				u = imageKey(u)
				if o, ok := objs[u]; ok {
					obj, covered = &o, true
					break
				}
				covered = covered || plain[u]
			}
			if !covered {
				// the page shows a picture twice (a carousel's clones, a thumbnail
				// beside it): the draft names it once
				if key := imageKey(urls[0]); seen[key] {
					return
				} else {
					seen[key] = true
				}
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
