package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(ImageObject{}) }

// ImageObject warns on a content image (one with alt text, outside the
// page's header, footer and navs, and not a product card's) that no
// ImageObject or image URL in the draft names. It needs the page's document,
// on a page search engines index.
type ImageObject struct{}

// ID is the check's ID.
func (ImageObject) ID() string { return "content/image-object" }

// Run reports each such image.
func (ImageObject) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	var out []audit.Issue
	for _, f := range imagesOf(c).unnamed {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's image %q has no ImageObject: add one (contentUrl, width, height, description from its alt text) as the image of the page or of the node it shows", f.alt))
	}
	return out
}
