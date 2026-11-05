package content

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Media{}) }

var (
	videoSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:[\w-]+\.)?(?:youtube(?:-nocookie)?\.com/(?:embed|watch|shorts)|youtu\.be/|player\.vimeo\.com/video/|vimeo\.com/\d|fast\.wistia\.(?:net|com)/embed|[\w-]+\.wistia\.(?:net|com)/(?:embed|medias)|player\.twitch\.tv|tiktok\.com/embed|dailymotion\.com/(?:embed|video)|dai\.ly/|loom\.com/(?:embed|share)|play\.vidyard\.com|[\w-]+\.hubs\.vidyard\.com|players\.brightcove\.net|[\w-]+\.jwplatform\.com|cdn\.jwplayer\.com/players|rumble\.com/embed|facebook\.com/plugins/video|fast\.wistia\.net|player\.mux\.com|iframe\.mediadelivery\.net|embed\.bunnycdn|streamable\.com/e/|player\.cloudinary\.com|video\.[\w-]+\.com/embed|e\.video-cdn\.net|fast\.vidalytics|share\.synthesia)`)
	audioSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:open\.spotify\.com/embed|w\.soundcloud\.com/player|embed\.podcasts\.apple\.com|anchor\.fm/.+/embed|podcasters\.spotify\.com/.+/embed|www\.buzzsprout\.com/.+\?.*player|[\w-]+\.buzzsprout\.com/.+/episodes|html5-player\.libsyn\.com|player\.libsyn\.com|[\w-]+\.podbean\.com/(?:e|player)|www\.podbean\.com/player|player\.simplecast\.com|player\.captivate\.fm|share\.transistor\.fm|player\.megaphone\.fm|omny\.fm/.+/embed|www\.spreaker\.com/embed|widget\.spreaker\.com|embed\.acast\.com|player\.rss\.com|player\.fireside\.fm|bandcamp\.com/EmbeddedPlayer|embed\.music\.apple\.com)`)

	// mediaClip is a file name that says the clip is the page's decoration.
	mediaClip = regexp.MustCompile(`(?i)(background|backdrop|[/_.-]bg[/_.-]|hero|ambient|loop)`)

	// socialCDN is where a social network keeps the clips of its posts.
	socialCDN = regexp.MustCompile(`(?i)//[\w.-]*(fbcdn\.net|cdninstagram\.com|twimg\.com|tiktokcdn[\w-]*\.com|scontent[\w.-]*)/`)
)

// videoID is the video a YouTube or Vimeo address names, "" for any other.
var videoID = regexp.MustCompile(`(?i)(?:youtube(?:-nocookie)?\.com/(?:embed/|shorts/|watch\?(?:[^#]*&)?v=)|youtu\.be/|vimeo\.com/(?:video/)?)([\w-]{6,})`)

// sameMedia reports two addresses of the same file, or of the same YouTube or
// Vimeo video.
func sameMedia(p *webpage.Facts, a, b string) bool {
	if p.Abs(a) == p.Abs(b) {
		return true
	}
	ma, mb := videoID.FindStringSubmatch(a), videoID.FindStringSubmatch(b)
	return ma != nil && mb != nil && ma[1] == mb[1]
}

// Media warns on a video or audio the page plays that no VideoObject or
// AudioObject has as its contentUrl or embedUrl. The VideoObject or
// AudioObject is what makes the media a search result of its own. It needs
// the page's document, on a page search engines index.
type Media struct{}

// ID is the check's ID.
func (Media) ID() string { return "content/media" }

// Run reports each player the draft has no object for.
func (Media) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p := c.Page
	var out []audit.Issue
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		kind, src := "", ""
		switch n.DataAtom {
		case atom.Video, atom.Audio:
			kind = "VideoObject"
			if n.DataAtom == atom.Audio {
				kind = "AudioObject"
			}
			src, _ = webpage.Attr(n, "src")
			if s := webpage.QueryAll(n, "source[src]"); src == "" && len(s) > 0 {
				src, _ = webpage.Attr(s[0], "src")
			}
			// a clip nobody can play or stop is decoration: it loops by itself, or
			// has no controls and is a background, a social post's or a script's
			_, controls := webpage.Attr(n, "controls")
			_, autoplay := webpage.Attr(n, "autoplay")
			_, loop := webpage.Attr(n, "loop")
			if !controls && (autoplay || loop || mediaClip.MatchString(src) || socialCDN.MatchString(src) || src == "") {
				return
			}
		case atom.Iframe:
			src, _ = webpage.Attr(n, "src")
			// a lazy loader keeps the player's address in a data attribute
			if lazy, _ := webpage.Attr(n, "data-src"); lazy != "" && (src == "" || strings.HasPrefix(src, "about:")) {
				src = lazy
			}
			switch {
			case videoSrc.MatchString(src):
				kind = "VideoObject"
			case audioSrc.MatchString(src):
				kind = "AudioObject"
			}
		}
		if kind == "" {
			return
		}
		for _, l := range c.Nodes {
			if !c.Is(l.N, kind) {
				continue
			}
			if src == "" {
				return
			}
			for _, k := range []string{"contentUrl", "embedUrl", "url"} {
				if slices.ContainsFunc(jsonld.Strs(l.N, k), func(s string) bool { return sameMedia(p, s, src) }) {
					return
				}
			}
		}
		what := src
		if what == "" {
			what = "a " + n.Data
		}
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, n), audit.Warning, "the page plays %s but no %s has it as contentUrl or embedUrl: add one (name, description, thumbnailUrl, uploadDate) linked from the WebPage", what, kind))
	})
	return out
}
