package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Model3D{}) }

// Model3D warns when a product page shows a 3D model of the product and the
// draft has no 3DModel: the model's file is the Product's subjectOf. It needs
// the page's document.
type Model3D struct{}

// ID is the check's ID.
func (Model3D) ID() string { return "product/3d-model" }

// Run reports the first 3D model the page embeds or links, at its element.
func (Model3D) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || c.Page.Doc == nil || c.Has("3DModel") {
		return nil
	}
	for _, n := range webpage.QueryAll(c.Page.Doc, `model-viewer, [src$=".glb"], [src$=".gltf"], [href$=".usdz"], [href$=".glb"]`) {
		src, _ := webpage.Attr(n, "src")
		if src == "" {
			src, _ = webpage.Attr(n, "href")
		}
		return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, n), audit.Warning, "the page shows a 3D model (%s): add a 3DModel (contentUrl, encodingFormat) as the Product's subjectOf", src)}
	}
	return nil
}
