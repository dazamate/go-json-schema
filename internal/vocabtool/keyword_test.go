package vocabtool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func issuesMatching(r Report, path, msg string) []Issue {
	p, m := regexp.MustCompile(path), regexp.MustCompile(msg)
	var out []Issue
	for _, i := range r.Issues {
		if p.MatchString(i.Path) && m.MatchString(i.Message) {
			out = append(out, i)
		}
	}
	return out
}

func checkDraft(t *testing.T, draft string, opts CheckOptions) Report {
	t.Helper()
	r, err := newTool(t).Check(json.RawMessage(draft), opts)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// keywordPage is a page whose nodes carry the keyword where the checker asks.
const keywordPage = `{"@context":"https://schema.org","@graph":[
 {"@id":"https://a.example/#org","@type":"OnlineBusiness","name":"Lift My Boxes","url":"https://a.example/"},
 {"@id":"https://a.example/#website","@type":"WebSite","name":"Lift My Boxes","url":"https://a.example/","publisher":{"@id":"https://a.example/#org"}},
 {"@id":"https://a.example/piano/#webpage","@type":"WebPage","url":"https://a.example/piano/",
  "name":"Cost to Move a Piano: Upright vs Grand","alternateName":"The cost to move a piano, from upright to grand",
  "description":"Get a free estimate of the cost to move a piano.","disambiguatingDescription":"See the cost to move a piano in the calculator below.",
  "keywords":["cost to move a piano"],"isPartOf":{"@id":"https://a.example/#website"},
  "hasPart":[
   {"@id":"https://a.example/piano/#article","@type":"Article","headline":"The Cost to Move a Piano","alternativeHeadline":"Cost to move a piano: upright vs grand","description":"The cost to move a piano, upright or grand.",
    "about":{"@id":"https://a.example/piano/#service","@type":"Service","name":"Piano moving","description":"Piano moving, and the cost to move a piano.","disambiguatingDescription":"The cost to move a piano with a crew."}},
   {"@id":"https://a.example/piano/#faq","@type":"FAQPage","name":"Cost to move a piano FAQ","mainEntity":[]},
   {"@id":"https://a.example/piano/#steps","@type":"HowTo","name":"Five ways to get a quote","description":"Check the cost to move a piano before you book."}],
  "about":[{"@id":"https://a.example/#piano","@type":"Thing","name":"Piano","description":"A keyboard instrument."}],
  "mentions":[{"@type":"MonetaryAmount","description":"Local move: upright, 10 road miles","currency":"USD","value":400}]}
]}`

func TestKeywordPassesAPageThatCarriesIt(t *testing.T) {
	r := checkDraft(t, keywordPage, CheckOptions{Keyword: "cost to move a piano", KeywordDensity: 1})
	for _, i := range r.Issues {
		if strings.Contains(i.Message, "keyword") {
			t.Errorf("%s %s: %s", i.Severity, i.Path, i.Message)
		}
	}
}

func TestKeywordFindsWhereItIsMissing(t *testing.T) {
	draft := strings.NewReplacer(
		`"alternateName":"The cost to move a piano, from upright to grand",`, ``,
		`"description":"Get a free estimate of the cost to move a piano."`, `"description":"Get a free estimate for your move."`,
		`"alternativeHeadline":"Cost to move a piano: upright vs grand",`, ``,
		`"name":"Cost to move a piano FAQ"`, `"name":"Piano moving FAQ"`,
		`"description":"Check the cost to move a piano before you book."`, `"description":"An estimate tells you what to expect. These steps make sure the final bill matches it, and that nothing on moving day surprises you or the crew, from the stairs to the parking and the tuning after the move."`,
		`"keywords":["cost to move a piano"]`, `"keywords":["piano movers"]`,
	).Replace(keywordPage)
	r := checkDraft(t, draft, CheckOptions{Keyword: "Cost to Move a Piano"})
	for _, want := range []struct{ path, msg string }{
		{`#?webpage|@graph\[2\]$`, `no alternateName: add one that carries the keyword "Cost to Move a Piano"`},
		{`\.description$`, `description "Get a free estimate for your move\." lacks the keyword`},
		{`hasPart\[0\]$`, `no alternativeHeadline`},
		{`hasPart\[1\]$`, `the FAQ's name lacks the keyword`},
		{`hasPart\[2\]\.description$`, `lacks the keyword`},
		{`@graph\[2\]$`, `keywords lacks the target keyword`},
	} {
		if len(issuesMatching(r, want.path, want.msg)) == 0 {
			t.Errorf("no issue at %s matching %q; got %v", want.path, want.msg, r.Issues)
		}
	}
	// The thing the page is about and the price it mentions describe something
	// else: the page's keyword would make them false.
	for _, i := range r.Issues {
		if strings.Contains(i.Path, ".about[0]") || strings.Contains(i.Path, ".mentions[") {
			t.Errorf("an entity or data node held to the keyword: %s %s", i.Path, i.Message)
		}
	}
}

func TestKeywordDensityOfTheMarkup(t *testing.T) {
	r := checkDraft(t, keywordPage, CheckOptions{Keyword: "cost to move a piano", KeywordDensity: 60})
	if len(issuesMatching(r, `^\$$`, `^the keyword "cost to move a piano" fills \d+\.\d% of the markup's words \(\d+ uses in \d+ words\), under the 60\.0% target`)) == 0 {
		t.Errorf("no markup density warning: %v", r.Issues)
	}
	if got := issuesMatching(checkDraft(t, keywordPage, CheckOptions{Keyword: "cost to move a piano"}), `^\$$`, `fills`); len(got) != 0 {
		t.Errorf("a markup over 3%%: %v", got)
	}
}

func TestKeywordDensityOfALongDescription(t *testing.T) {
	long := "The cost to move a piano " + strings.Repeat("depends on the stairs and the distance ", 20) // 1 use in 146 words
	draft := strings.Replace(keywordPage, `"Check the cost to move a piano before you book."`, `"`+long+`"`, 1)
	r := checkDraft(t, draft, CheckOptions{Keyword: "cost to move a piano"})
	if len(issuesMatching(r, `hasPart\[2\]\.description$`, `gives the keyword "cost to move a piano" 3\.4% of its words`)) != 0 {
		t.Fatalf("5 words of 146 is 3.4%%, over 3%%: %v", r.Issues)
	}
	r = checkDraft(t, draft, CheckOptions{Keyword: "cost to move a piano", KeywordDensity: 5})
	if len(issuesMatching(r, `hasPart\[2\]\.description$`, `3\.4% of its words \(1 use in 146 words\), under the 5\.0% target`)) == 0 {
		t.Errorf("no density warning at 5%%: %v", r.Issues)
	}
}

func TestKeywordSkipsNodesOtherPagesShare(t *testing.T) {
	draft := strings.Replace(keywordPage, `"@id":"https://a.example/piano/#steps"`, `"@id":"https://a.example/#calculator"`, 1)
	draft = strings.Replace(draft, `"Check the cost to move a piano before you book."`, `"A free calculator."`, 1)
	r := checkDraft(t, draft, CheckOptions{Keyword: "cost to move a piano"})
	if got := issuesMatching(r, `hasPart\[2\]`, `keyword`); len(got) != 0 {
		t.Errorf("a node with a site-level @id was held to one page's keyword: %v", got)
	}
}

func TestNoKeywordNoKeywordChecks(t *testing.T) {
	r := checkDraft(t, keywordPage, CheckOptions{})
	if got := issuesMatching(r, ``, `keyword`); len(got) != 0 {
		t.Errorf("keyword checks without a keyword: %v", got)
	}
}

func TestOrganizationsInAboutAndMentions(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "lmb", "how-much-does-a-moving-company-cost.jsonld"))
	if err != nil {
		t.Fatal(err)
	}
	r := checkDraft(t, string(b), CheckOptions{})
	for _, want := range []string{
		`^PODS \(company\) is another business in about`,
		`^U-Haul is another business in about`,
		`^United States Postal Service is an organization in about: .*move it to mentions`,
	} {
		if len(issuesMatching(r, `\.about\[\d+\]$`, want)) == 0 {
			t.Errorf("no issue matching %q", want)
		}
	}
	if got := issuesMatching(r, ``, `^Lift My Boxes is`); len(got) != 0 {
		t.Errorf("the site's own business flagged: %v", got)
	}

	// The keyword names the business: it is the page's subject.
	r = checkDraft(t, string(b), CheckOptions{Keyword: "U-Haul truck rental cost"})
	if got := issuesMatching(r, ``, `^U-Haul is`); len(got) != 0 {
		t.Errorf("a business the keyword names flagged: %v", got)
	}

	// The home page is about the site's own business.
	b, err = os.ReadFile(filepath.Join("testdata", "lmb", "home.jsonld"))
	if err != nil {
		t.Fatal(err)
	}
	if got := issuesMatching(checkDraft(t, string(b), CheckOptions{}), ``, `in about|in mentions`); len(got) != 0 {
		t.Errorf("home page: %v", got)
	}
}

func TestBusinessInMentions(t *testing.T) {
	draft := `{"@context":"https://schema.org","@graph":[
 {"@id":"https://a.example/#org","@type":"OnlineBusiness","name":"Lift My Boxes"},
 {"@id":"https://a.example/#website","@type":"WebSite","publisher":{"@id":"https://a.example/#org"}},
 {"@id":"https://a.example/p/#webpage","@type":"WebPage","url":"https://a.example/p/","mentions":[
  {"@type":"Organization","name":"U-Haul","additionalType":["https://www.wikidata.org/wiki/Q4830453"]},
  {"@type":"GovernmentOrganization","name":"Federal Motor Carrier Safety Administration"},
  {"@type":"NewsMediaOrganization","name":"CNET"},
  {"@type":"Organization","name":"Los Angeles Metro","additionalType":["https://www.wikidata.org/wiki/Q7835189"]}]}]}`
	r := checkDraft(t, draft, CheckOptions{})
	if len(issuesMatching(r, `mentions\[0\]$`, `^U-Haul is another business in mentions: .*never a competitor`)) == 0 {
		t.Errorf("no warning on a business in mentions: %v", r.Issues)
	}
	if got := issuesMatching(r, `mentions\[[123]\]$`, `in mentions`); len(got) != 0 {
		t.Errorf("an agency, a publisher or a transit district flagged as a business: %v", got)
	}
}

func TestLooseNodes(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "lmb", "cost-to-move-a-piano.jsonld"))
	if err != nil {
		t.Fatal(err)
	}
	r := checkDraft(t, string(b), CheckOptions{})
	got := issuesMatching(r, `^\$$`, `^\d+ top-level nodes are referenced from other nodes \(.*Piano.*\): validator\.schema\.org lists`)
	if len(got) != 1 {
		t.Fatalf("loose-node warnings = %v, want one that names Piano", got)
	}
	if strings.Contains(got[0].Message, "Lift My Boxes,") {
		t.Errorf("the business is named loose: %s", got[0].Message)
	}

	// Nested where first referenced, nothing is loose.
	if got := issuesMatching(checkDraft(t, keywordPage, CheckOptions{}), ``, `top-level nodes`); len(got) != 0 {
		t.Errorf("a nested graph: %v", got)
	}
}
