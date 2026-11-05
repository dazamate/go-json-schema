package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestImageObjectFields(t *testing.T) {
	const body = `<img src="/img/tea-640.webp" alt="A couple serving tea to elders" width="640" height="427">`
	bare := graph(`{"@type":"ImageObject","url":"https://x.test/img/tea-640.webp"}`)
	run(t, content.ImageObjectFields{}, append([]tc{
		{"an ImageObject without its size or description", at(bare, body), warn("$.@graph[1]", `this ImageObject is the page's image "A couple serving tea to elders": set width 640, height 427, a description from its alt text`)},
		{"an ImageObject without its description", at(graph(`{"@type":"ImageObject","url":"https://x.test/img/tea-640.webp","width":640,"height":427}`), body), warn("$.@graph[1]", `this ImageObject is the page's image "A couple serving tea to elders": set a description from its alt text`)},
		{"an ImageObject with all three", at(graph(`{"@type":"ImageObject","url":"https://x.test/img/tea-640.webp","width":640,"height":427,"caption":"A couple serving tea"}`), body), nil},
		// no ImageObject at all is the other check's
		{"no ImageObject", at(webPage, body), nil},
		{"an image that gives no size", at(bare, `<img src="/img/tea-640.webp" alt="A couple serving tea to elders">`), warn("$.@graph[1]", `this ImageObject is the page's image "A couple serving tea to elders": set a description from its alt text`)},
		textOnly(bare, "A couple serving tea to elders"),
	}, hidden(bare, body)...))
}
