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
		// players of other hosts
		{"a Loom video", at(webPage, `<iframe src="https://www.loom.com/embed/8a1b2c3d" title="Demo"></iframe>`), warn("html:#main > iframe", vo("https://www.loom.com/embed/8a1b2c3d", "VideoObject"))},
		{"a Dailymotion video", at(webPage, `<iframe src="https://www.dailymotion.com/embed/video/x7tgad0"></iframe>`), warn("html:#main > iframe", vo("https://www.dailymotion.com/embed/video/x7tgad0", "VideoObject"))},
		{"a Vidyard video", at(webPage, `<iframe src="https://play.vidyard.com/Tn4k2Ll.html"></iframe>`), warn("html:#main > iframe", vo("https://play.vidyard.com/Tn4k2Ll.html", "VideoObject"))},
		{"a Wistia video on the account's host", at(webPage, `<iframe src="https://acme.wistia.com/embed/iframe/abc123"></iframe>`), warn("html:#main > iframe", vo("https://acme.wistia.com/embed/iframe/abc123", "VideoObject"))},
		{"a Facebook video", at(webPage, `<iframe src="https://www.facebook.com/plugins/video.php?href=https%3A%2F%2Ffb.com%2Fv%2F1"></iframe>`), warn("html:#main > iframe", vo("https://www.facebook.com/plugins/video.php?href=https%3A%2F%2Ffb.com%2Fv%2F1", "VideoObject"))},
		{"a podcast on Libsyn", at(webPage, `<iframe src="https://html5-player.libsyn.com/embed/episode/id/123"></iframe>`), warn("html:#main > iframe", vo("https://html5-player.libsyn.com/embed/episode/id/123", "AudioObject"))},
		{"a lazy YouTube iframe", at(webPage, `<iframe src="about:blank" data-src="https://www.youtube.com/embed/abc123"></iframe>`), warn("html:#main > iframe", vo("https://www.youtube.com/embed/abc123", "VideoObject"))},
		{"a lazy YouTube iframe the draft names", at(withVideo, `<iframe data-src="https://www.youtube.com/embed/abc123"></iframe>`), nil},
		{"the draft names the video by its watch address", at(graph(`{"@type":"VideoObject","name":"Tea","embedUrl":"https://www.youtube.com/watch?v=abc123"}`), video), nil},
		// media that is not the page's content
		{"a looping background video", at(webPage, `<video class="embed-responsive-item" muted loop playsinline autoplay><source src="/media/OPUS_Summer.mp4"></video>`), nil},
		{"a footer's background video", at(webPage, `<video playsinline src="/assets/images/nw-footer-background.mp4?rev=0a4d"></video>`), nil},
		{"a player an ad script feeds", at(webPage, `<video class="raptive-player-video" playsinline poster="https://cdn.jwplayer.com/v2/media/88q5T96f/poster.jpg"></video>`), nil},
		{"a social feed's clip", at(webPage, `<video poster="" src="https://instagram.fbne7-1.fna.fbcdn.net/o1/v/t16/f2/m84/AQO.mp4?_nc_cat=108" class="instagram-gallery-item__media"></video>`), nil},
		textOnly(webPage, "Our tea ceremony"),
	}, hidden(webPage, video)...))
}
