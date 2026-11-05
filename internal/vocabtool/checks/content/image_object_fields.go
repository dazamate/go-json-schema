package content

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ImageObjectFields{}) }

// ImageObjectFields warns on an ImageObject for a content image that lacks
// the image's width or height, or a description. It says each ImageObject
// once. It needs the page's document, on a page search engines index.
type ImageObjectFields struct{}

// ID is the check's ID.
func (ImageObjectFields) ID() string { return "content/image-object-fields" }

// Run reports each such ImageObject.
func (ImageObjectFields) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	var out []audit.Issue
	for _, f := range imagesOf(c).incomplete {
		out = append(out, audit.Issuef(f.path, audit.Warning, "this ImageObject is the page's image %q: set %s", f.alt, strings.Join(f.miss, ", ")))
	}
	return out
}
