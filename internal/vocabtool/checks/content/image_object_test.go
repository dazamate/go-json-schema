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
		textOnly(webPage, "A couple serving tea to elders"),
	}, hidden(webPage, body)...))
}
