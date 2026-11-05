package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

// The footers are trimmed from real pages: a Cincinnati dentist (Yelp), a
// Vancouver hotel (TripAdvisor, its address doubled), an NZ pest controller
// (NoCowboys) and an Auckland car dealer's header (Google Maps).
func TestReviewProfile(t *testing.T) {
	const url = "https://clinic.test/fees/"
	page := func(body string) string {
		return `<!doctype html><html><head><title>Fees</title></head><body><main><h1>Fees</h1><p>Our fees.</p>` + body + `</main></body></html>`
	}
	yelp := `<html><body><main><h1>Fees</h1><p>Our fees.</p></main><footer><h5>Connect</h5><ul class="social">
<li><a aria-label="Follow us on Facebook" title="Follow us on Facebook" href="https://www.facebook.com/MooreheadDentistry" target="_blank"><i class="fab fa-facebook-f"></i></a></li>
<li><a aria-label="Read Yelp Reviews" title="Read Yelp Reviews" href="https://www.yelp.com/biz/moorehead-family-dentistry-cincinnati" target="_blank"><i class="fab fa-yelp"></i></a></li></ul></footer></body></html>`
	hotel := `<html><body><main><h1>Fees</h1><p>Our fees.</p></main><footer><div class="flat-menu footer_social no_heading"><ul class="nav"><li class=""><a target="_blank" title="Trip Advisor" href="http://www.tripadvisor.com/http://www.tripadvisor.com/Hotel_Review-g154943-d224689-Reviews-Opus_Hotel-Vancouver_British_Columbia.html"><i class="fa fa-tripadvisor fa-2x" aria-hidden="true"></i><span class="sr-only">Trip Advisor</span></a></li>
<li><a href="https://www.tripadvisor.com/TravelersChoice-Hotels-cTop-g153339">Travelers' Choice</a></li></ul></div></footer></body></html>`
	pest := `<html><body><main><h1>Fees</h1><p>Our fees.</p></main><div id="footer"><section class="fusion-footer-widget-column widget widget_text"><div class="textwidget"><table><tbody><tr><td align="left"><a href="http://www.nocowboys.co.nz/businesses/Pestproof-Pest-Control" target="_blank" rel="noopener"><img alt="Pestproof On No Cowboys" src="/nocowboys.png"></a></td></tr></tbody></table></div></section></div></body></html>`
	maps := `<html><body><header><div class="info"><div class="addr"><span class="content"><a target="_blank" href="https://www.google.com/maps/place/10+Doncaster+Street+Mangere+Auckland+nz">10 Doncaster Street, Mangere, Auckland</a></span></div></div></header><main><h1>Fees</h1></main></body></html>`
	listicle := page(`<h2>Best dentists in Cincinnati</h2><ul><li><a href="https://www.yelp.com/biz/acme-dental-cincinnati">Acme Dental</a></li><li><a href="https://www.yelp.com/biz/smile-works-cincinnati">Smile Works</a></li></ul>`)
	review := page(`<p>Happy with us? <a href="https://www.yelp.com/biz/moorehead-family-dentistry-cincinnati">Review us on Yelp</a>.</p>`)
	contact := page(`<div class="contact-details"><p>1521 Walmart Drive</p><a href="https://g.page/moorehead-dentistry-lebanon?share">View Map</a></div>`)
	held := `{"@context":"https://schema.org","@type":"Dentist","name":"Moorehead","sameAs":["https://www.facebook.com/MooreheadDentistry","https://www.yelp.com/biz/moorehead-family-dentistry-cincinnati"]}`
	bare := `{"@context":"https://schema.org","@type":"Dentist","name":"Moorehead"}`
	ask := `that no sameAs lists: add them to the business's sameAs`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a Yelp page in a footer", audittest.Input{Draft: bare, PageHTML: yelp, PageURL: url}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the business's profiles on review sites (yelp.com/biz/moorehead-family-dentistry-cincinnati) ` + ask},
		}},
		{"a doubled TripAdvisor address, not the Travelers' Choice page", audittest.Input{Draft: bare, PageHTML: hotel, PageURL: url}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the business's profiles on review sites (tripadvisor.com/Hotel_Review-g154943-d224689-Reviews-Opus_Hotel-Vancouver_British_Columbia.html) ` + ask},
		}},
		{"an image link in a footer widget", audittest.Input{Draft: bare, PageHTML: pest, PageURL: url}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the business's profiles on review sites (nocowboys.co.nz/businesses/Pestproof-Pest-Control) ` + ask},
		}},
		{"a Google Maps place in the header", audittest.Input{Draft: bare, PageHTML: maps, PageURL: url}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the business's profiles on review sites (google.com/maps/place/10+Doncaster+Street+Mangere+Auckland+nz) ` + ask},
		}},
		{"a link worded as the business's own", audittest.Input{Draft: bare, PageHTML: review, PageURL: url}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the business's profiles on review sites (yelp.com/biz/moorehead-family-dentistry-cincinnati) ` + ask},
		}},
		{"a Google place in a contact block", audittest.Input{Draft: bare, PageHTML: contact, PageURL: url}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links the business's profiles on review sites (g.page/moorehead-dentistry-lebanon?share) ` + ask},
		}},
		{"a list of other businesses' profiles", audittest.Input{Draft: bare, PageHTML: listicle, PageURL: url}, nil},
		{"sameAs lists it", audittest.Input{Draft: held, PageHTML: yelp, PageURL: url}, nil},
		{"a noindex page", audittest.Input{Draft: bare, PageHTML: audittest.Noindex(yelp), PageURL: url}, nil},
		{"a page known by its text", audittest.Input{Draft: bare, PageText: "Our fees. Review us on Yelp.", PageURL: url}, nil},
		{"no page", audittest.Input{Draft: bare, PageURL: url}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.ReviewProfile{}, tc.in, tc.want) })
	}
}
