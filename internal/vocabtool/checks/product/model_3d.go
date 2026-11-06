package product

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
)

func init() { audit.Register(Model3D{}) }

// Model3D warns when a product page shows a 3D model of the product and the
// draft has no 3DModel: the model's file is the Product's subjectOf. It needs
// the page's document.
type Model3D struct{}

// ID is the check's ID.
func (Model3D) ID() string { return "product/model-3d" }

var (
	// modelFile is the address of a 3D model's file, a CDN's version after it
	// included: /ring.glb, /chair.usdz?v=17.
	modelFile = regexp.MustCompile(`(?i)\.(?:glb|gltf|usdz)(?:[?#]\S*)?$`)
	// viewControl names a control that shows the model: "View in 3D", "Show
	// 3D modal", "View in your space", "See it in your room", "View in AR";
	// a room is the model's only after "in" ("See room ideas" is a link).
	viewControl = regexp.MustCompile(`(?i)^\s*(?:view|see|show)(?: (?:it|this|item|product|model))?(?: (?:3d|ar|augmented reality)\b| (?:in|on)(?: (?:your|my|a))? (?:3d|ar|augmented reality|space|room|home)\b)`)
)

// Run reports the first 3D model the page shows: the model's file, a viewer
// frame the page lets use WebXR, or a control that opens the model.
func (Model3D) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || c.Page.Doc == nil || c.Has("3DModel") {
		return nil
	}
	n, shown := modelShown(c.Page.Doc)
	if n == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, n), audit.Warning, "the page shows a 3D model (%s): add a 3DModel (the model file's contentUrl and encodingFormat, or its viewer's embedUrl) as the Product's subjectOf", shown)}
}

// modelShown is the element that shows a 3D model and what it shows: the
// model's address, or the control's name in quotes. A file or viewer comes
// before a control, as it gives the model's address.
func modelShown(doc *html.Node) (*html.Node, string) {
	var file, frame, control *html.Node
	var fileURL, frameURL, name string
	webpage.Visit(doc, func(n *html.Node) {
		if n.Type != html.ElementNode || file != nil {
			return
		}
		switch n.Data {
		case "model-viewer":
			src, _ := webpage.Attr(n, "src")
			file, fileURL = n, src
			return
		case "iframe":
			if allow, _ := webpage.Attr(n, "allow"); frame == nil && strings.Contains(allow, "xr-spatial-tracking") {
				frame = n
				frameURL, _ = webpage.Attr(n, "src")
			}
		case "a", "button":
			if control == nil {
				if t := strings.Join(strings.Fields(webpage.HTMLText(n)), " "); viewControl.MatchString(t) {
					control, name = n, fmt.Sprintf("%q", t)
				} else if l, _ := webpage.Attr(n, "aria-label"); viewControl.MatchString(l) {
					control, name = n, fmt.Sprintf("%q", strings.TrimSpace(l))
				}
			}
		}
		for _, a := range n.Attr {
			if (a.Key == "src" || a.Key == "href") && modelFile.MatchString(a.Val) {
				file, fileURL = n, a.Val
				return
			}
		}
	})
	switch {
	case file != nil:
		if fileURL == "" {
			fileURL = "model-viewer"
		}
		return file, fileURL
	case frame != nil:
		return frame, frameURL
	}
	return control, name
}
