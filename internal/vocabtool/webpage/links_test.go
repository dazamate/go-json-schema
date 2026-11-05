package webpage

import "testing"

func TestSocialProfileTakesProfilesOnly(t *testing.T) {
	for _, h := range []string{
		"https://www.youtube.com/watch?v=abc123",
		"https://www.youtube.com/shorts/abc123",
		"https://twitter.com/acme/status/1234567890",
		"https://x.com/acme/status/1234567890",
		"https://www.instagram.com/p/Cx1234/",
		"https://www.instagram.com/reel/Cx1234/",
		"https://www.facebook.com/acme/posts/1234567890",
		"https://www.facebook.com/sharer/sharer.php?u=https://acme.test",
		"https://www.facebook.com/share/abc/",
		"https://www.tiktok.com/@acme/video/123",
		"https://www.pinterest.com/pin/123456/",
		"https://www.linkedin.com/posts/acme_launch-123",
		"https://www.etsy.com/listing/123/ring",
	} {
		if SocialProfile(h) {
			t.Errorf("%s is no profile", h)
		}
	}
	for _, h := range []string{
		"https://www.youtube.com/@acme",
		"https://www.youtube.com/channel/UC12345",
		"https://www.youtube.com/c/acme",
		"https://www.youtube.com/user/acme",
		"https://x.com/acme",
		"https://www.instagram.com/acme/",
		"https://www.facebook.com/acme",
		"https://www.facebook.com/profile.php?id=100012345",
		"https://www.tiktok.com/@acme",
		"https://www.linkedin.com/company/acme",
		"https://www.linkedin.com/in/jane-smith",
		"https://www.pinterest.com/acme/",
		"https://www.etsy.com/shop/acme",
		"https://bsky.app/profile/acme.bsky.social",
	} {
		if !SocialProfile(h) {
			t.Errorf("%s is a profile", h)
		}
	}
}
