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

func TestReviewProfileTakesBusinessPagesOnly(t *testing.T) {
	for _, h := range []string{
		"https://www.yelp.com",
		"https://www.yelp.com/search?find_desc=dentist",
		"https://www.yelp.com/c/cincinnati/dentists",
		"https://www.tripadvisor.com/TravelersChoice-Hotels-cTop-g153339",
		"https://www.tripadvisor.com/Hotels-g154943-Vancouver-Hotels.html",
		"https://www.checkatrade.com/search?cat=plumber",
		"https://www.checkatrade.com/trades",
		"https://www.trustpilot.com/",
		"https://www.trustpilot.com/categories/plumber",
		"https://www.google.com/maps/dir//10+Doncaster+Street,+Mangere/@-36.97,174.77",
		"https://www.google.com/maps/search/plumbers",
		// autodiscount.co.nz's footer: a map of its street address, its hasMap
		"https://www.google.com/maps/place/10+Doncaster+Street+Mangere+Auckland+nz",
		"https://www.google.com/search?q=acme",
		"https://www.bbb.org/search?find_text=acme",
		"https://www.booking.com/index.html",
		"https://www.booking.com/searchresults.html?ss=Vancouver",
		"https://www.houzz.com/professionals/general-contractors/c/Boston",
		"https://www.facebook.com/AcmePlumbing",
		"/yelp.com/biz/acme",
	} {
		if ReviewProfile(h) {
			t.Errorf("%s is no business's profile", h)
		}
	}
	for _, h := range []string{
		"https://www.yelp.com/biz/moorehead-family-dentistry-cincinnati",
		"https://www.yelp.com.au/biz/acme-plumbing-sydney",
		"https://www.yelp.ca/biz/acme-toronto?osq=plumber",
		"http://www.tripadvisor.com/http://www.tripadvisor.com/Hotel_Review-g154943-d224689-Reviews-Opus_Hotel-Vancouver_British_Columbia.html",
		"https://www.tripadvisor.co.nz/Restaurant_Review-g255106-d1234-Reviews-Acme-Auckland.html",
		"https://www.checkatrade.com/trades/jamesplumbingandheatinglimited",
		"https://www.trustpilot.com/review/acme.example",
		"https://uk.trustpilot.com/review/acme.example",
		"https://www.google.com/maps/place/Auto+Discount+Mangere/@-36.97,174.77,17z",
		"https://www.google.com/maps/place/7-Eleven/@40.7,-74.0,17z",
		"https://www.google.co.uk/maps/place/Acme+Plumbing/@51.5,-0.1,17z",
		"https://www.google.com/maps?cid=11399163827163773228",
		"https://g.page/moorehead-dentistry-lebanon?share",
		"https://g.page/r/CabcDEF/review",
		"https://goo.gl/maps/7pFmrok3avE2cSCS7",
		"https://maps.app.goo.gl/AbCd123",
		"http://www.nocowboys.co.nz/businesses/Pestproof-Pest-Control",
		"https://www.productreview.com.au/listings/acme-plumbing",
		"https://www.houzz.com/pro/acme-builders/acme-builders-pfvwus-pf~123",
		"https://www.yell.com/biz/acme-plumbing-london-123456/",
		"https://www.hipages.com.au/connect/acmeplumbing",
		"https://www.bbb.org/us/oh/cincinnati/profile/dentist/acme-dental-0292-123",
		"https://www.healthgrades.com/physician/dr-jane-smith-abc12",
		"https://www.zocdoc.com/practice/acme-dental-12345",
		"https://www.opentable.com/r/acme-bistro-auckland",
		"https://www.booking.com/hotel/ca/opus-vancouver.html",
		"https://www.trustatrader.com/traders/james-plumbing-and-heating-ltd-plumbers-tw2-tw4-tw13",
	} {
		if !ReviewProfile(h) {
			t.Errorf("%s is a business's profile", h)
		}
	}
}
