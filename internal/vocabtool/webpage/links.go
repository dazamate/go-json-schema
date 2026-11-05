package webpage

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var (
	socialHost  = regexp.MustCompile(`(?i)^(?:www\.|m\.)?(instagram\.com|facebook\.com|fb\.com|tiktok\.com|pinterest\.[a-z.]+|youtube\.com|x\.com|twitter\.com|linkedin\.com|threads\.net|etsy\.com|snapchat\.com|bsky\.app)$`)
	socialSkips = regexp.MustCompile(`(?i)/(?:sharer|intent|share|pin/create|sharing|dialog|plugins)\b|privacy|polic(?:y|ies)|terms|legal|/(?:help|about|settings|login|tos)(?:/|$)`)

	// ShareLink matches the address of a share button: a social network's
	// sharing dialog, not a link to the business's own profile.
	ShareLink = regexp.MustCompile(`(?i)facebook\.com/sharer|(?:twitter|x)\.com/(?:intent|share)|pinterest\.[a-z.]+/pin/create|linkedin\.com/(?:sharing|shareArticle)|wa\.me/\?|api\.whatsapp\.com/send|reddit\.com/submit`)
)

// profilePrefixes are the first segments of a profile address with a second
// segment (/company/acme, /channel/UC1, /shop/acme); socialPages are single
// segments that are a network's own pages, not an account's name.
var (
	profilePrefixes = []string{"company", "school", "showcase", "in", "channel", "c", "user", "shop", "profile", "add"}
	socialPages     = []string{"watch", "playlist", "results", "shorts", "feed", "home", "explore", "search", "hashtag", "p", "reel", "reels", "tv", "stories", "status", "i", "pin", "posts", "videos", "live", "groups", "events", "marketplace", "gaming", "login", "signup", "business", "ads", "pulse", "jobs", "messaging", "notifications", "watch.php", "photo.php", "photo", "story.php", "permalink.php"}
)

// SocialLink is a link to a social network: a profile, a post, a video or a
// share button.
func SocialLink(href string) bool {
	u, err := url.Parse(strings.TrimSpace(href))
	return err == nil && u.IsAbs() && socialHost.MatchString(u.Hostname())
}

// SocialProfile is a link to one of the business's social profiles: a single
// path segment (/@name, /name), or a profile prefix and its name (/company/name,
// /channel/id, /c/name, /user/name). A video, a post, a tweet or a pin is not.
func SocialProfile(href string) bool {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil || !u.IsAbs() || !socialHost.MatchString(u.Hostname()) || socialSkips.MatchString(u.Path) {
		return false
	}
	segs := Segments(u)
	switch len(segs) {
	case 1:
		return !slices.Contains(socialPages, segs[0])
	case 2:
		return slices.Contains(profilePrefixes, segs[0])
	}
	return false
}
