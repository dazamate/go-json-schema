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

// reviewProfiles are the addresses of a business's own page on a review or
// listing site: the host, then the path that names one business (a listing's
// /biz/ or /trades/ segment, a hotel's _Review- page), not the site's home,
// search or category pages.
var googleHost = regexp.MustCompile(`(?i)^(?:www\.)?google\.[a-z.]+$`)

var reviewProfiles = []struct{ host, path *regexp.Regexp }{
	{regexp.MustCompile(`(?i)^(?:[a-z]+\.)?yelp\.[a-z.]+$`), regexp.MustCompile(`(?i)^/biz/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:[a-z]+\.)?tripadvisor\.[a-z.]+$`), regexp.MustCompile(`(?i)/(?:Hotel|Restaurant|Attraction|VacationRentalReview|Attraction_Product)_?Review-[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?checkatrade\.com$`), regexp.MustCompile(`(?i)^/trades/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:[a-z]+\.)?trustpilot\.com$`), regexp.MustCompile(`(?i)^/review/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.|maps\.)?google\.[a-z.]+$`), regexp.MustCompile(`(?i)^/maps/place/[^/]+`)},
	{regexp.MustCompile(`(?i)^g\.page$`), regexp.MustCompile(`^/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:maps\.app\.)?goo\.gl$`), regexp.MustCompile(`(?i)^/(?:maps/)?[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?nocowboys\.co\.nz$`), regexp.MustCompile(`(?i)^/businesses/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?productreview\.com\.au$`), regexp.MustCompile(`(?i)^/listings/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?houzz\.[a-z.]+$`), regexp.MustCompile(`(?i)^/pro/[^/]+|-pf~\d+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?yell\.com$`), regexp.MustCompile(`(?i)^/biz/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?hipages\.com\.au$`), regexp.MustCompile(`(?i)^/connect/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?bbb\.org$`), regexp.MustCompile(`(?i)/profile/[^/]+/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?healthgrades\.com$`), regexp.MustCompile(`(?i)^/(?:physician|dentist|group-directory|hospital-directory)/[^/]+|^/providers/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?zocdoc\.com$`), regexp.MustCompile(`(?i)^/(?:practice|doctor|dentist)/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?opentable\.[a-z.]+$`), regexp.MustCompile(`(?i)^/(?:r|restaurant/profile)/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?booking\.com$`), regexp.MustCompile(`(?i)^/hotel/[a-z]{2}/[^/]+`)},
	{regexp.MustCompile(`(?i)^(?:www\.)?trustatrader\.com$`), regexp.MustCompile(`(?i)^/traders/[^/]+`)},
}

// ReviewProfile is a link to one business's own page on a review or listing
// site (a Yelp /biz/ page, a Checkatrade trade, a Google Maps place, a
// TripAdvisor hotel): the page its owner puts in sameAs. A site's home, search
// and category pages are not.
func ReviewProfile(href string) bool { return ReviewProfileAddress(href) != "" }

// ReviewProfileAddress is the address a ReviewProfile link names, or "". A
// page can double a link ("tripadvisor.com/http://tripadvisor.com/Hotel_Review-
// ..."): the last address in it counts.
func ReviewProfileAddress(href string) string {
	href = strings.TrimSpace(href)
	if i := strings.Index(href[min(len(href), 8):], "/http"); i >= 0 {
		href = href[min(len(href), 8)+i+1:]
	}
	u, err := url.Parse(href)
	if err != nil || !u.IsAbs() {
		return ""
	}
	for _, r := range reviewProfiles {
		if r.host.MatchString(u.Hostname()) && r.path.MatchString(u.Path) {
			return href
		}
	}
	// a map link by its cid names one place
	if googleHost.MatchString(u.Hostname()) && u.Path == "/maps" && u.Query().Get("cid") != "" {
		return href
	}
	return ""
}
