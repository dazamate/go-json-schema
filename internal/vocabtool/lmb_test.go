package vocabtool

import (
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// The Lift My Boxes pages under testdata/lmb are a content site's, not a
// shop's: moving-cost guides with FAQs, calculators, data modules, rules and
// checklists. Each was trimmed from the dev site as of 2026-10-31, with the
// JSON-LD it carried then in the .jsonld beside it. lmbTests mirror the
// acceptance tests of the schema-mcp gap review run on those pages
// (marketing-research/schema-mcp-gaps/acceptance.py): what build_jsonld must
// and must not say on them.

const lmbSD = "how-much-do-movers-cost-in-san-diego"

// lmbTest is one check: the page, the draft ("bare": a WebPage with the
// page's url and name; "current": the page's own JSON-LD), and an issue whose
// path and message match (want) or none that does (never).
type lmbTest struct {
	id, kind, page, mode, path, message string
}

var lmbTests = []lmbTest{
	// what the checker reported before the review
	{"C1", "want", lmbSD, "bare", ``, `no WPHeader`},
	{"C2", "want", lmbSD, "bare", ``, `breadcrumb <nav> has no BreadcrumbList`},
	{"C3", "want", lmbSD, "bare", ``, `copyrightYear`},
	{"C4", "want", lmbSD, "bare", ``, `no Table`},
	{"C5", "want", "contact-us", "bare", ``, `subtype ContactPage`},

	// types the page calls for
	{"G01", "want", lmbSD, "bare", ``, `no FAQPage`},
	{"G02", "want", lmbSD, "bare", ``, `the draft has no WebSite`},
	{"G03", "want", lmbSD, "bare", ``, `no node publishes the page`},
	{"G04", "want", "home", "current", ``, `type it OnlineBusiness`},
	{"G05", "want", "moving-cost-calculator-live", "bare", ``, `but no WebApplication`},
	{"G06", "want", lmbSD, "bare", `^html:#calculator`, `has no QuoteAction`},
	{"G07", "want", lmbSD, "bare", ``, `no node is an Article`},
	{"G08", "want", lmbSD, "bare", ``, `no Dataset describes it`},
	{"G09", "want", lmbSD, "bare", ``, `add an Observation`},
	{"G10", "want", lmbSD, "bare", ``, `estimatedSalary`},
	{"G11", "want", lmbSD, "bare", ``, `as a UnitPriceSpecification`},
	{"G12", "want", lmbSD, "bare", ``, `as a CompoundPriceSpecification`},
	{"G13", "want", lmbSD, "bare", ``, `as a MonetaryAmount\b`},
	{"G14", "want", "how-much-do-movers-cost", "bare", ``, `add a Product`},
	{"G15", "want", "cost-to-move-a-piano", "bare", ``, `no Service describes it`},
	{"G16", "want", "cheapest-long-distance-moving", "bare", ``, `add a Legislation`},
	{"G17", "want", "boston-movers-cost", "bare", ``, `add a GovernmentPermit`},
	{"G18", "want", lmbSD, "bare", ``, `as a GovernmentOrganization`},
	{"G19", "want", "how-much-do-movers-cost", "bare", ``, `"In this guide"`},
	{"G20", "want", lmbSD, "bare", `countdown`, `add a HowTo\b`},
	{"G21", "want", lmbSD, "bare", ``, `that no citation names`},
	{"G22", "want", "seattle-movers-cost", "bare", ``, `but no node has that date`},
	{"G23", "want", lmbSD, "current", ``, `contactOption TollFree`},
	{"G24", "want", "how-much-do-movers-cost-in-philadelphia", "current", ``, `Q21010817: type it City`},
	{"G25", "want", "cost-of-moving-to-nyc", "current", ``, `Q20857065: type it GovernmentOrganization`},
	{"G26", "want", lmbSD, "current", ``, `Q131427665: type it AdministrativeArea`},
	{"G27", "want", lmbSD, "current", ``, `Q245016: type it DefenceEstablishment`},
	{"G28", "want", lmbSD, "current", ``, `Q62078547: type it CollegeOrUniversity`},
	{"G29", "want", "how-much-does-a-moving-company-cost", "current", ``, `Q1757263: type it MovingCompany`},
	{"G30", "want", "how-much-do-movers-cost-bay-area", "current", ``, `Q7814330: type it Bridge`},

	// optional fits
	{"B01", "want", lmbSD, "bare", `figure|weather`, `no ImageObject`},
	{"B02", "want", lmbSD, "bare", ``, `SpeakableSpecification|\babstract\b`},
	{"B03", "want", "2-movers-cost", "bare", `truck-picker`, `as a Vehicle|add a Vehicle`},
	{"B04", "want", "cheapest-long-distance-moving-options", "bare", ``, `DefinedTerm`},
	{"B05", "want", "how-much-do-movers-cost", "bare", ``, `CollectionPage`},
	{"B06", "want", lmbSD, "bare", ``, `QuantitativeValue`},

	// Search Console's Product snippets report on the live calculator page, 2026-11-04
	{"P01", "want", "moving-cost-calculator-live", "current", `itemListElement`, `critical: Either "offers", "review", or "aggregateRating" should be specified\. This node is a Vehicle`},
	{"P02", "want", "moving-cost-calculator-live", "current", `\.offers$`, `non-critical: Missing field "offerCount" \(in "offers"\)`},
	{"P03", "want", "moving-cost-calculator-live", "current", ``, `non-critical: Missing field "review"; Missing field "aggregateRating"`},
	{"P04", "want", "2-movers-cost", "bare", `truck-picker`, `marks one without offers, review or aggregateRating invalid`},
	{"P05", "want", "moving-cost-calculator-live", "bare", ``, `AggregateOffer \(lowPrice, highPrice, priceCurrency, and offerCount`},

	// what the checker must not say
	{"M1", "never", lmbSD, "bare", `Key takeaways|calc|#calculator`, `WPSideBar`},
	{"M2", "never", lmbSD, "bare", `tip-calculator-who`, `form has no Action`},
	{"M3", "never", "2-movers-cost", "bare", ``, `counts 2 items`},
	{"M4", "never", "cheapest-long-distance-moving", "current", ``, `subtype HowToItem`},
	{"M5", "never", lmbSD, "bare", ``, `as an Offer's price or, for a range, an AggregateOffer's`},
	{"M6", "never", "how-much-do-movers-cost-houston", "bare", `search`, `form has no Action`},
	{"M7", "never", "how-much-do-movers-cost-in-los-angeles", "bare", ``, `counts 10 items`},
	{"M8", "never", "2-movers-cost", "current", ``, `Q783794: type it Organization`},
}

var (
	lmbCanonical = regexp.MustCompile(`<link rel="canonical" href="([^"]+)"`)
	lmbTitle     = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	lmbReports   sync.Map // page + " " + mode: Report
)

// lmbReport checks a Lift My Boxes page as the review did: no page_url, and
// either a bare WebPage or the page's own JSON-LD as the draft.
func lmbReport(t *testing.T, page, mode string) Report {
	t.Helper()
	if r, ok := lmbReports.Load(page + " " + mode); ok {
		return r.(Report)
	}
	b, err := os.ReadFile(filepath.Join("testdata", "lmb", page+".html"))
	if err != nil {
		t.Fatal(err)
	}
	var draft []byte
	if mode == "current" {
		if draft, err = os.ReadFile(filepath.Join("testdata", "lmb", page+".jsonld")); err != nil {
			t.Fatal(err)
		}
	} else {
		bare := map[string]string{"@context": "https://schema.org", "@type": "WebPage", "url": "", "name": ""}
		if m := lmbCanonical.FindSubmatch(b); m != nil {
			bare["url"] = string(m[1])
		}
		if m := lmbTitle.FindSubmatch(b); m != nil {
			bare["name"] = html.UnescapeString(string(m[1]))
		}
		draft, _ = json.Marshal(bare)
	}
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{PageHTML: string(b)})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	lmbReports.Store(page+" "+mode, r)
	return r
}

func TestLMBAcceptance(t *testing.T) {
	for _, tc := range lmbTests {
		t.Run(tc.id, func(t *testing.T) {
			r := lmbReport(t, tc.page, tc.mode)
			path, msg := regexp.MustCompile(tc.path), regexp.MustCompile(tc.message)
			var hits []Issue
			for _, i := range r.Issues {
				if path.MatchString(i.Path) && msg.MatchString(i.Message) {
					hits = append(hits, i)
				}
			}
			switch {
			case tc.kind == "want" && len(hits) == 0:
				t.Errorf("no issue on %s (%s) at /%s/ saying /%s/", tc.page, tc.mode, tc.path, tc.message)
			case tc.kind == "never" && len(hits) > 0:
				t.Errorf("%s (%s): %s: %s", tc.page, tc.mode, hits[0].Path, hits[0].Message)
			}
		})
	}
}

// lmbPages are the Lift My Boxes fixtures.
func lmbPages(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "lmb", "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range files {
		out = append(out, strings.TrimSuffix(filepath.Base(f), ".html"))
	}
	return out
}

// A content site that sells nothing gets none of the shop's asks, and no ask
// its markup already meets.
func TestLMBStaysQuiet(t *testing.T) {
	never := []struct{ mode, message string }{
		{"", `itemCondition`},                               // "damaged item" in a moving guide
		{"", `business is based in`},                        // "movers based in the five boroughs"
		{"", `shows how to pay`},                            // "pay by bank transfer in advance"
		{"", `free shipping|shipping rate|return policy`},   // no shop policies
		{"", `this WebPage is an? \w+ page: type it`},       // the subtype check already asks
		{"", `defines \d+ terms \((Includes|How long|Who)`}, // labels each move option repeats
		{"current", `no FAQPage|no Question carries`},       // its FAQPage holds the questions
	}
	for _, page := range lmbPages(t) {
		for _, mode := range []string{"bare", "current"} {
			r := lmbReport(t, page, mode)
			for _, n := range never {
				if n.mode != "" && n.mode != mode {
					continue
				}
				re := regexp.MustCompile(n.message)
				for _, i := range r.Issues {
					if re.MatchString(i.Message) {
						t.Errorf("%s (%s): %s: %s", page, mode, i.Path, i.Message)
					}
				}
			}
		}
	}
}

func TestSocialProfileLeavesPolicyPagesOut(t *testing.T) {
	for _, h := range []string{"https://twitter.com/en/privacy", "https://www.facebook.com/full_data_use_policy", "https://www.instagram.com/legal/terms/", "https://www.youtube.com/about/policies/"} {
		if webpage.SocialProfile(h) {
			t.Errorf("%s is a profile", h)
		}
	}
	for _, h := range []string{"https://www.instagram.com/queen_of_diamonds_jewellery/", "https://www.facebook.com/104992909320900", "https://www.linkedin.com/company/acme", "https://www.youtube.com/@acme", "https://x.com/acme"} {
		if !webpage.SocialProfile(h) {
			t.Errorf("%s is not a profile", h)
		}
	}
}

func TestCheckLeavesToolAndSummaryAsidesOutOfSidebars(t *testing.T) {
	page := `<!doctype html><html><body><main><article><h1>Movers</h1>
<section id="calc"><div><label>From <input id="from"></label><label>To <input id="to"></label>
<aside class="calc-note"><p>Short moves are billed by the hour.</p></aside></div></section>
<section><h2>Key takeaways</h2><aside aria-label="Key takeaways"><ul><li>Book early.</li></ul></aside></section>
<aside><h2>Related guides</h2><p>Packing</p></aside>
</article></main></body></html>`
	r := checkPage(t, `{"@context":"https://schema.org","@type":"WebPage","url":"https://x.test/","name":"Movers"}`, page)
	var asides []string
	for _, e := range r.PageElements {
		if e.Type == "WPSideBar" {
			asides = append(asides, e.Selector)
		}
	}
	if len(asides) != 1 || !strings.HasSuffix(asides[0], "article > aside") {
		t.Errorf("WPSideBars %v, want only the related guides", asides)
	}
}

func TestCheckLeavesDialogAndFilterFormsAlone(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Movers</h1>
<dialog id="tips"><form method="dialog"><button>Close</button></form></dialog>
<search><form id="zip-filter"><input name="zip" placeholder="ZIP"><button>Filter</button></form></search>
<div role="search"><form id="site-search" action="/search/"><input name="q"><button>Search</button></form></div>
</main></body></html>`
	r := checkPage(t, `{"@context":"https://schema.org","@type":"WebPage","url":"https://x.test/movers/","name":"Movers"}`, page)
	if n := issueCount(r, "form has no Action"); n != 0 {
		t.Errorf("%d form warnings on a dialog's close and the search forms: %+v", n, r.Issues)
	}
	if !warns(r, "search box", "SearchAction") {
		t.Errorf("the site search lost its SearchAction ask: %+v", r.Issues)
	}
}

func TestCheckCountsAHeadingOnlyWhenItsSectionHoldsTheItems(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>What do 2 movers cost?</h1><p>Two movers cost $100 an hour.</p>
<ul><li>Crew</li><li>Truck</li><li>Fuel</li></ul>
<h2>Moving two bedrooms</h2><p>A two-bedroom move takes a day.</p>
<h2>Order LADOT tow-away signs 10 business days before the move</h2><p>Signs go up early.</p>
<h2>Five tips for moving day</h2><h3>Pack early</h3><p>a</p><h3>Label boxes</h3><p>b</p><h3>Book a lift</h3><p>c</p><h3>Eat well</h3><p>d</p><h3>Rest</h3><p>e</p>
<h2>Three questions to ask</h2><details><summary>Is it binding?</summary><p>Yes.</p></details><details><summary>Who packs?</summary><p>They do.</p></details><details><summary>Insured?</summary><p>Yes.</p></details>
</main></body></html>`
	r := checkPage(t, `{"@context":"https://schema.org","@type":"WebPage","url":"https://x.test/","name":"Movers"}`, page)
	for _, no := range []string{"counts 2 items", "counts 10 items"} {
		if n := issueCount(r, no); n != 0 {
			t.Errorf("%q: %+v", no, r.Issues)
		}
	}
	expectWarnings(t, r, [][]string{{"Five tips", "counts 5 items"}, {"Three questions", "counts 3 items"}})
}

func TestCheckLeavesBreadcrumbItemsTyped(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
		{"@type":"ListItem","position":1,"name":"Home","item":"https://x.test/"},
		{"@type":"ListItem","position":2,"name":"How to get the cheapest long distance moving quote","item":"https://x.test/how-to-get-a-quote/"}]}`)
	if n := issueCount(r, "names the subtype"); n != 0 {
		t.Errorf("a breadcrumb item was retyped: %+v", r.Issues)
	}
	r = check(t, `{"@context":"https://schema.org","@type":"ListItem","name":"How to pack a tool kit"}`)
	if n := issueCount(r, "HowToItem"); n != 0 {
		t.Errorf("a how-to title named HowToItem: %+v", r.Issues)
	}
}

func TestCheckKeepsAnAsideBesideAFormASidebar(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Contact</h1><div class="grid">
<form method="post" action="/contact/"><input name="name"><input name="email"><textarea name="msg"></textarea></form>
<aside aria-label="Other ways to reach us"><p>Call us.</p></aside></div></main></body></html>`
	r := checkPage(t, `{"@context":"https://schema.org","@type":"ContactPage","url":"https://x.test/contact/","name":"Contact"}`, page)
	if !slices.ContainsFunc(r.PageElements, func(e PageElement) bool { return e.Type == "WPSideBar" }) {
		t.Errorf("no WPSideBar: %+v", r.PageElements)
	}
}

func TestCheckNamesTouristAttractionForALandmark(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@type":"Thing","name":"Alamo Mission","additionalType":["https://www.wikidata.org/wiki/Q1564373"]}`)
	if !warns(r, "Q1564373: type it LandmarksOrHistoricalBuildings", "TouristAttraction") {
		t.Errorf("no landmark typing: %+v", r.Issues)
	}
}
