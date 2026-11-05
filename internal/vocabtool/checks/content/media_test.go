package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestMedia(t *testing.T) {
	const video = `<iframe src="https://www.youtube.com/embed/abc123" title="Our tea ceremony"></iframe>`
	vo := func(what, kind string) string {
		return "the page plays " + what + " but no " + kind + " has it as contentUrl or embedUrl: add one (name, description, thumbnailUrl, uploadDate) linked from the WebPage"
	}
	withVideo := graph(`{"@type":"VideoObject","name":"Our tea ceremony","embedUrl":"https://www.youtube.com/embed/abc123"}`)
	run(t, content.Media{}, append([]tc{
		{"a video no VideoObject has", at(webPage, video), warn("html:#main > iframe", vo("https://www.youtube.com/embed/abc123", "VideoObject"))},
		{"an audio element no AudioObject has", at(webPage, `<audio controls></audio>`), warn("html:#main > audio", vo("a audio", "AudioObject"))},
		{"a video element's source", at(webPage, `<video><source src="/tea.mp4"></video>`), warn("html:#main > video", vo("/tea.mp4", "VideoObject"))},
		{"a VideoObject with its embedUrl", at(withVideo, video), nil},
		{"an iframe that is no player", at(webPage, `<iframe src="https://maps.test/embed"></iframe>`), nil},
		textOnly(webPage, "Our tea ceremony"),
	}, hidden(webPage, video)...))
}
