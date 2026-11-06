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
			{Path: "html:body > main > model-viewer", Severity: audit.Warning, Message: `the page shows a 3D model (/ring.glb): add a 3DModel (the model file's contentUrl and encodingFormat, or its viewer's embedUrl) as the Product's subjectOf`},
		}},
		// a CDN adds a version to the file's address
		{"a model file with a query", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", `<a href="/models/ring.glb?v=1700000000" download>Download the model</a>`), PageURL: productURL}, []audit.Issue{
			{Path: "html:body > main > a", Severity: audit.Warning, Message: `the page shows a 3D model (/models/ring.glb?v=1700000000): add a 3DModel (the model file's contentUrl and encodingFormat, or its viewer's embedUrl) as the Product's subjectOf`},
		}},
		// ikea.com: the button loads the model when pressed
		{"a button that opens the model", audittest.Input{Draft: noDraft, PageHTML: productPageWith("DYVLINGE", "IKEA", `<button type="button" data-skapa="button@19.0.27" aria-label="Show 3D modal" class="pipf-btn pipf-btn--small pipf-leading-icon pipf-btn--image-overlay pipf-xr-button pipf-xr-ga-main pipf-xr-button--desktop"><span class="pipf-typography-label-s pipf-btn__inner"><svg viewBox="0 0 24 24" focusable="false"><path d="M12 2"></path></svg><span class="pipf-btn__label">View in 3D</span></span></button>`), PageURL: productURL}, []audit.Issue{
			{Path: `html:button[aria-label="Show 3D modal"]`, Severity: audit.Warning, Message: `the page shows a 3D model ("View in 3D"): add a 3DModel (the model file's contentUrl and encodingFormat, or its viewer's embedUrl) as the Product's subjectOf`},
		}},
		{"a link to see it in a room", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", `<a href="#ar">View in your space</a>`), PageURL: productURL}, []audit.Issue{
			{Path: "html:body > main > a", Severity: audit.Warning, Message: `the page shows a 3D model ("View in your space"): add a 3DModel (the model file's contentUrl and encodingFormat, or its viewer's embedUrl) as the Product's subjectOf`},
		}},
		// boho.ng: a viewer the frame lets use WebXR, as Sketchfab's embeds do
		{"a viewer frame allowed WebXR", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Tubi table", "Boho", `<div data-swiftxr-viewer-block="true" data-swiftxr-url="https://boho.swiftxr.site/tubi-diy-table-boho"><iframe data-swiftxr-viewer-iframe="" title="SwiftXR viewer" src="https://boho.swiftxr.site/tubi-diy-table-boho" width="100%" height="100%" loading="lazy" frameborder="0" allow="fullscreen; autoplay; camera; microphone; gyroscope; accelerometer; magnetometer; xr-spatial-tracking; web-share" allowfullscreen=""></iframe></div>`), PageURL: productURL}, []audit.Issue{
			{Path: "html:body > main > div > iframe", Severity: audit.Warning, Message: `the page shows a 3D model (https://boho.swiftxr.site/tubi-diy-table-boho): add a 3DModel (the model file's contentUrl and encodingFormat, or its viewer's embedUrl) as the Product's subjectOf`},
		}},
		{"a video frame", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", `<iframe src="https://www.youtube.com/embed/abc" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"></iframe>`), PageURL: productURL}, nil},
		{"a 3D printing note", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", `<a href="/pages/how-we-make-it">See how we 3D print the wax</a>`), PageURL: productURL}, nil},
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
