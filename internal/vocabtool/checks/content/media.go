package content

import (
	"regexp"
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Media{}) }

var (
	videoSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:www\.)?(?:youtube(?:-nocookie)?\.com/(?:embed|watch|shorts)|youtu\.be/|player\.vimeo\.com/video/|vimeo\.com/\d|fast\.wistia\.(?:net|com)/embed|player\.twitch\.tv|www\.tiktok\.com/embed)`)
	audioSrc = regexp.MustCompile(`(?i)^(?:https?:)?//(?:open\.spotify\.com/embed|w\.soundcloud\.com/player|embed\.podcasts\.apple\.com|anchor\.fm/.+/embed|podcasters\.spotify\.com/.+/embed|www\.buzzsprout\.com/.+\?.*player)`)
)

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
		case atom.Iframe:
			src, _ = webpage.Attr(n, "src")
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
				if slices.ContainsFunc(jsonld.Strs(l.N, k), func(s string) bool { return p.Abs(s) == p.Abs(src) }) {
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
