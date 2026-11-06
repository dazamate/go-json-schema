package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestImageObject(t *testing.T) {
	const body = `<img src="/img/tea-640.webp" alt="A couple serving tea to elders" width="640" height="427">`
	const msg = `the page's image "A couple serving tea to elders" has no ImageObject: add one (contentUrl, width, height, description from its alt text) as the image of the page or of the node it shows`
	const logo = `<header id="site-header"><a href="/"><img src="/logo.png" alt="Ann Lee Weddings logo" width="40" height="40"></a></header>`
	run(t, content.ImageObject{}, append([]tc{
		{"an image no node names", at(webPage, body), warn("html:#main > img", msg)},
		{"an image named by an ImageObject", at(graph(`{"@type":"ImageObject","url":"https://x.test/img/tea-640.webp"}`), body), nil},
		{"an image named by a node's image", at(graph(`{"@type":"Article","headline":"Tea","image":"https://x.test/img/tea-640.webp"}`), body), nil},
		{"an image named by its srcset", at(graph(`{"@type":"ImageObject","url":"https://x.test/img/tea-320.webp"}`), `<img src="/other.webp" srcset="/img/tea-320.webp 320w" alt="A couple serving tea to elders">`), nil},
		{"an image with no alt text", at(webPage, `<img src="/img/tea-640.webp" alt="">`), nil},
		{"a tracking pixel", at(webPage, `<img src="/pixel.gif" alt="Pixel" width="1" height="1">`), nil},
		{"the page header's logo", at(webPage, logo), nil},
		{"a product card's image", contactPage(at(webPage, `<h2>You may also like</h2><div class="card"><a href="/p/gold/"><img src="/gold.webp" alt="Gold ring"></a><span>$120.00</span></div><div class="card"><a href="/p/silver/"><img src="/silver.webp" alt="Silver ring"></a><span>$90.00</span></div>`)), nil},
		// a lazy-loading page: the placeholder is no image, the real one is in data-src
		{"a lazy image the draft names", at(graph(`{"@type":"ImageObject","url":"https://x.test/thmb/Bdf8n6.jpg"}`), `<img src="data:image/gif;base64,R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==" data-src="/thmb/Bdf8n6.jpg" alt="chocolate chip cookies on a plate">`), nil},
		{"a lazy image the draft lacks", at(webPage, `<img src="data:," data-src="/thmb/Bdf8n6.jpg" alt="chocolate chip cookies on a plate">`), warn("html:#main > img", `the page's image "chocolate chip cookies on a plate" has no ImageObject: add one (contentUrl, width, height, description from its alt text) as the image of the page or of the node it shows`)},
		{"a lazy image with data-srcset the draft names", at(graph(`{"@type":"ImageObject","url":"https://x.test/thmb/big.jpg"}`), `<img src="data:," data-srcset="/thmb/small.jpg 300w, /thmb/big.jpg 1200w" alt="top down view of all ingredients">`), nil},
		{"a picture's source the draft names", at(graph(`{"@type":"ImageObject","url":"https://x.test/img/fig-tart.webp"}`), `<picture><source srcset="/img/fig-tart.webp 544w"><img src="data:," alt="Jamie drizzling honey on top of a fig tart"></picture>`), nil},
		{"an image with only a placeholder", at(webPage, `<img src="data:," alt="a photo of the beach">`), nil},
		{"one image shown twice", at(webPage, `<div id="gallery"><img src="/img/tea-640.webp" alt="A couple serving tea to elders"><img src="/img/tea-640.webp" alt="A couple serving tea to elders"></div>`), warn("html:#gallery > img:nth-of-type(1)", msg)},
		{"a resized copy the draft names", at(graph(`{"@type":"ImageObject","url":"https://x.test/cdn/files/jack.jpg"}`), `<img src="/cdn/files/jack_533x.jpg?v=1710&width=533" alt="flatlay of Jack in Blue">`), nil},
		{"a WordPress size the draft names", at(graph(`{"@type":"ImageObject","url":"https://x.test/wp-content/uploads/tea.jpg"}`), `<img src="/wp-content/uploads/tea-300x200.jpg" alt="A couple serving tea to elders">`), nil},
		// pictures that are not the page's content
		{"a partner's logo", at(webPage, `<img src="/img/google.svg" alt="Google logo"><img src="/img/openai.svg" alt="OpenAI logo">`), nil},
		{"an arrow", at(webPage, `<img src="/img/l.svg" alt="left arrow"><img src="/img/r.svg" alt="right arrow">`), nil},
		{"an icon of the page's script", at(webPage, `<img src="/img/x.png" alt="opens in a new window"><img src="/img/c.png" alt="Close"><img src="/img/m.png" alt="navIcon"><img src="/img/i.png" alt="icon">`), nil},
		{"an emoji", at(webPage, `<img src="/emoji/1f4de.svg" alt="📞"><img src="/emoji/1f4c5.svg" alt="📅">`), nil},
		{"a missing image's stand-in", at(webPage, `<img src="/img/none.png" alt="No image available">`), nil},
		// a thumbnail that links to another page is that page's picture
		{"a teaser's thumbnail", at(webPage, `<a href="/recipes/new/"><img src="/img/pasta.jpg" alt="New recipe! Pasta salad"></a>`), nil},
		{"a category tile of another site", at(webPage, `<a href="https://youtube.com/@x"><img src="/img/yt.jpg" alt="YouTube Recipes"></a>`), nil},
		{"an image that links to its larger copy", at(webPage, `<a href="/img/tea-1200.webp"><img src="/img/tea-640.webp" alt="A couple serving tea to elders"></a>`), warn("html:#main > a > img", msg)},
		{"an image that links to the top of the page", at(webPage, `<a href="#"><img src="/img/tea-640.webp" alt="A couple serving tea to elders"></a>`), warn("html:#main > a > img", msg)},
		{"an image that links to the page", at(webPage, `<a href="/tea/"><img src="/img/tea-640.webp" alt="A couple serving tea to elders"></a>`), warn("html:#main > a > img", msg)},
		{"the controls of a gallery", at(webPage, `<img src="/i/p.svg" alt="previous image"><img src="/i/w.svg" alt="removed from Wishlist"><img src="/i/s.svg" alt="Share this product"><img src="/i/z.svg" alt="Zoom">`), nil},
		{"a tracking pixel with a URL for alt text", at(webPage, `<img src="/i/t.gif" alt="Leads https-www.jamieoliver.com-recipes">`), nil},
		{"an icon named by its file", at(webPage, `<img src="/img/q.png" alt="High_quality_materials_icon_1"><img src="/media/noun_sanitizer_3513506_v2.max-150x150.png" alt="Sanitizer Updated"><img src="/static/icons/wifi.svg" alt="Wifi Updated">`), nil},
		{"a gallery thumbnail", at(webPage, `<img src="/i/aqua-t1.jpg" alt="2015 Toyota Aqua - Thumbnail">`), nil},
		{"a flag on a button", at(webPage, `<button type="submit"><img src="/flags/gg.svg" alt="Guernsey"></button>`), nil},
		textOnly(webPage, "A couple serving tea to elders"),
	}, hidden(webPage, body)...))
}
