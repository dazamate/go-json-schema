package webpage

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	socialHost  = regexp.MustCompile(`(?i)^(?:www\.|m\.)?(instagram\.com|facebook\.com|fb\.com|tiktok\.com|pinterest\.[a-z.]+|youtube\.com|x\.com|twitter\.com|linkedin\.com|threads\.net|etsy\.com|snapchat\.com|bsky\.app)$`)
	socialSkips = regexp.MustCompile(`(?i)/(?:sharer|intent|share|pin/create|sharing|dialog|plugins)\b|privacy|polic(?:y|ies)|terms|legal|/(?:help|about|settings|login|tos)(?:/|$)`)

	// ShareLink matches the address of a share button: a social network's
	// sharing dialog, not a link to the business's own profile.
	ShareLink = regexp.MustCompile(`(?i)facebook\.com/sharer|(?:twitter|x)\.com/(?:intent|share)|pinterest\.[a-z.]+/pin/create|linkedin\.com/(?:sharing|shareArticle)|wa\.me/\?|api\.whatsapp\.com/send|reddit\.com/submit`)
)

// SocialProfile is a link to one of the business's social profiles.
func SocialProfile(href string) bool {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil || !u.IsAbs() || !socialHost.MatchString(u.Hostname()) || socialSkips.MatchString(u.Path) {
		return false
	}
	return strings.Trim(u.Path, "/") != ""
}
