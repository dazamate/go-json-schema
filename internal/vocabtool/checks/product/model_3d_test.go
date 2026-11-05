package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestModel3D(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", `<model-viewer src="/ring.glb"></model-viewer>`)
	carried := `{"@context":"https://schema.org","@type":"3DModel","contentUrl":"https://harbour.test/ring.glb"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a 3D model", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body > main > model-viewer", Severity: audit.Warning, Message: `the page shows a 3D model (/ring.glb): add a 3DModel (contentUrl, encodingFormat) as the Product's subjectOf`},
		}},
		{"a 3DModel is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"no 3D model", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: `Art Deco ring`, PageURL: productURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Model3D{}, tc.in, tc.want) })
	}
}
