package research

import (
	"encoding/json"
	"errors"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/textrazor"
)

// calibrationURL is where the spec's calibration run read our page.
const calibrationURL = "http://localhost:8080/moving-cost-calculator/"

func selectFixturePath(name string) string { return filepath.Join("testdata", "select", name) }

func readSelectFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(selectFixturePath(name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

var (
	htmlTitle = regexp.MustCompile(`(?s)<title[^>]*>(.*?)</title>`)
	htmlLang  = regexp.MustCompile(`<html[^>]*\slang="([^"]*)"`)
	htmlH1    = regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1>`)
	htmlLink  = regexp.MustCompile(`(?s)<a\b[^>]*?\shref="([^"]*)"[^>]*>(.*?)</a>`)
	htmlTag   = regexp.MustCompile(`<[^>]*>`)
)

// htmlText is an element's text: its tags stripped, entities unescaped and
// whitespace collapsed.
func htmlText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(htmlTag.ReplaceAllString(s, ""))), " ")
}

// calibrationPage is the calibration page as pageJS reads it: page.html's
// title, lang, first H1 and <main> links, with hrefs resolved against base,
// and main.txt's text without its first line, which is the title.
func calibrationPage(t *testing.T, base string) rawPage {
	t.Helper()
	page := string(readSelectFixture(t, "page.html"))
	b, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	title, lang, h1 := htmlTitle.FindStringSubmatch(page), htmlLang.FindStringSubmatch(page), htmlH1.FindStringSubmatch(page)
	start, end := strings.Index(page, "<main"), strings.Index(page, "</main>")
	if title == nil || lang == nil || h1 == nil || start < 0 || end < start {
		t.Fatal("page.html lacks a <title>, <html lang>, <h1> or <main>")
	}
	_, text, _ := strings.Cut(string(readSelectFixture(t, "main.txt")), "\n")
	raw := rawPage{Title: htmlText(title[1]), Lang: lang[1], Headings: []string{"h1: " + htmlText(h1[1])}, Text: text}
	for _, m := range htmlLink.FindAllStringSubmatch(page[start:end], -1) {
		href, err := b.Parse(html.UnescapeString(m[1]))
		if err != nil {
			t.Fatal(err)
		}
		raw.Links = append(raw.Links, pageLink{Href: href.String(), Text: htmlText(m[2])})
	}
	return raw
}

// wikiFixture is wiki.json: the Wikipedia and Wikidata answers of the
// calibration run, by QID and title. Its fields are in key order, so the
// file's keys are sorted.
type wikiFixture struct {
	Items     map[string]json.RawMessage `json:"items"`     // wbgetentities entity by QID, as returned
	Links     map[string][]string        `json:"links"`     // page -> pages it links to, among those asked about
	Options   map[string][]string        `json:"options"`   // disambiguation page -> the pages it links to
	Pages     map[string]fixturePage     `json:"pages"`     // title -> what a title lookup found
	Redirects map[string]string          `json:"redirects"` // from -> to
}

type fixturePage struct {
	Disambig bool   `json:"disambig,omitempty"`
	Missing  bool   `json:"missing,omitempty"`
	QID      string `json:"qid,omitempty"`
}

// loadWikiFixture builds a fake Wikipedia from wiki.json, skipping the test
// until it is recorded.
func loadWikiFixture(t *testing.T) *fakeWiki {
	t.Helper()
	b, err := os.ReadFile(selectFixturePath("wiki.json"))
	if errors.Is(err, fs.ErrNotExist) {
		t.Skip("record wiki.json first: SCHEMA_MCP_RECORD=1 go test ./internal/research -run RecordSelect")
	}
	if err != nil {
		t.Fatal(err)
	}
	var fx wikiFixture
	if err := json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	w := &fakeWiki{pages: map[string]wikiPage{}, redirects: map[string][2]string{}, rawItems: fx.Items, articleLinks: fx.Links}
	for title, p := range fx.Pages {
		if !p.Missing {
			w.pages[title] = wikiPage{qid: p.QID, disambig: p.Disambig, links: fx.Options[title]}
		}
	}
	for from, to := range fx.Redirects {
		w.redirects[from] = [2]string{to, ""}
	}
	return w
}

// calibrationResearcher is a Researcher on the browser with the calibration
// run's TextRazor answer for any text and its cosines for each theme.
func calibrationResearcher(t *testing.T, b Browser) *Researcher {
	t.Helper()
	var tr struct {
		Response struct {
			Entities []textrazor.Entity `json:"entities"`
			Topics   []textrazor.Topic  `json:"topics"`
		} `json:"response"`
	}
	if err := json.Unmarshal(readSelectFixture(t, "textrazor.json"), &tr); err != nil {
		t.Fatal(err)
	}
	var cos struct {
		Keyword string                                     `json:"keyword"`
		Themes  map[string]struct{ Keyword, Page float64 } `json:"themes"`
	}
	if err := json.Unmarshal(readSelectFixture(t, "cosines.json"), &cos); err != nil {
		t.Fatal(err)
	}
	if cos.Keyword != calibrationQuery().Keyword {
		t.Fatalf("cosines.json is for %q", cos.Keyword)
	}
	scorer := &fakeScorer{cos: map[string][2]float64{}}
	for label, c := range cos.Themes {
		scorer.cos[label] = [2]float64{c.Keyword, c.Page}
	}
	return &Researcher{
		Browser: b,
		Linker: &fakeLinker{link: func(string) (textrazor.Result, error) {
			return textrazor.Result{Entities: tr.Response.Entities, Topics: tr.Response.Topics, Units: 1}, nil
		}},
		Themes: scorer,
	}
}

func calibrationQuery() SelectQuery {
	return SelectQuery{URL: calibrationURL, Keyword: "moving cost calculator", Settings: DefaultSelectSettings}
}

func findSelected(sel []Selected, name string) Selected {
	for _, s := range sel {
		if s.Name == name {
			return s
		}
	}
	return Selected{}
}

// TestSelectEntitiesCalibration replays the spec's calibration run on the
// moving cost calculator page from the recorded fixtures.
func TestSelectEntitiesCalibration(t *testing.T) {
	w := loadWikiFixture(t)
	raw := calibrationPage(t, calibrationURL)
	rep := runSelect(t, calibrationResearcher(t, fakeBrowser{selectPage(w, raw, nil)}), calibrationQuery())
	// A theme label missing from cosines.json fails the scoring, which the
	// tool reports as a note.
	if strings.Contains(rep.Note, "no themes") {
		t.Fatalf("note = %q", rep.Note)
	}
	defer func() {
		if b, err := json.MarshalIndent(rep, "", "  "); t.Failed() && err == nil {
			t.Logf("report: %s", b)
		}
	}()

	wantAbout := []string{"Calculator", "Moving company", "Cost", "Relocation (personal)", "Relocation service", "Transport",
		"Freight transport", "Business economics", "Pricing", "Land transport", "Accounting"}
	if got := selNames(rep.About); !slices.Equal(got, wantAbout) {
		t.Errorf("about = %v\nwant    %v", got, wantAbout)
	}
	wantMentions := []string{"Truck", "Stairs", "ZIP Code", "Federal Motor Carrier Safety Administration", "Weight", "Piano", "Hot tub", "Billiard table"}
	if got := selNames(rep.Mentions); !slices.Equal(got, wantMentions) {
		t.Errorf("mentions = %v\nwant       %v", got, wantMentions)
	}
	if hot := findSelected(rep.Mentions, "Hot tub"); !slices.Contains(hot.QualifiedBy, "scores") {
		t.Errorf("hot tub qualified_by = %v", hot.QualifiedBy)
	}
	pool := findSelected(rep.Mentions, "Billiard table")
	if !slices.Equal(pool.QualifiedBy, []string{"link"}) || !slices.Equal(pool.LinkedTo, []string{"http://localhost:8080/cost-to-move-a-pool-table/"}) {
		t.Errorf("billiard table qualified_by %v, linked_to %v", pool.QualifiedBy, pool.LinkedTo)
	}

	if d, ok := findDropped(rep, "Email"); !ok || d.Reason != ReasonBelowRelevance {
		t.Errorf("email dropped = %+v", d)
	}
	if d, ok := findDropped(rep, "Computing"); !ok || d.Reason != ReasonBelowSimilarity || num(d.KeywordSimilarity) != 0.409 || num(d.PageSimilarity) != 0.492 {
		t.Errorf("computing dropped = %+v", d)
	}

	for _, s := range slices.Concat(rep.About, rep.Mentions) {
		if want := s.Source == SourcePage; s.KGMIDChecked != want {
			t.Errorf("%s (%s): kgmid_checked %v, want %v", s.Name, s.Source, s.KGMIDChecked, want)
		}
	}

	looks := map[string]KeywordLookup{}
	for _, l := range rep.KeywordLookups {
		looks[l.Phrase] = l
	}
	if m := looks["moving"]; m.Result != "disambiguation" || !slices.Equal(m.Chosen, []string{"Relocation (personal)", "Relocation service"}) || m.Links != 2 {
		t.Errorf("moving lookup = %+v", m)
	}
	for phrase, want := range map[string]string{"cost": "exact", "calculator": "exact", "moving cost": "missing", "cost calculator": "missing", "moving cost calculator": "missing"} {
		if got := looks[phrase].Result; got != want {
			t.Errorf("%s lookup = %+v, want %s", phrase, looks[phrase], want)
		}
	}
	if len(looks) != 6 {
		t.Errorf("lookups = %+v", rep.KeywordLookups)
	}

	if rep.Words != len(strings.Fields(raw.Text)) || rep.Language != "en" || rep.Title != raw.Title {
		t.Errorf("words %d, language %q, title %q", rep.Words, rep.Language, rep.Title)
	}

	m := thingJSON(t, pool.Thing)
	if _, ok := m["@id"]; ok || m["name"] != "Billiard table" || m["@type"] != "Thing" {
		t.Errorf("billiard table thing = %s", pool.Thing)
	}
	if got := anyStrs(m["alternateName"]); !slices.Equal(got, []string{"pool tables", "Pool table"}) {
		t.Errorf("alternateName = %v", got)
	}
	wantSame := []string{
		"https://en.wikipedia.org/wiki/Billiard_table",
		"https://www.wikidata.org/wiki/Q3500725",
		"https://www.google.com/search?kgmid=/m/04p0qw",
		"http://dbpedia.org/resource/Billiard_table",
	}
	if got := anyStrs(m["sameAs"]); !slices.Equal(got, wantSame) {
		t.Errorf("sameAs = %v", got)
	}
	var ids [][2]string
	list, _ := m["identifier"].([]any)
	for _, x := range list {
		pv, _ := x.(map[string]any)
		id, _ := pv["propertyID"].(string)
		v, _ := pv["value"].(string)
		ids = append(ids, [2]string{id, v})
	}
	if want := [][2]string{{"Wikidata", "Q3500725"}, {"GoogleMachineID", "kg:/m/04p0qw"}}; !slices.Equal(ids, want) {
		t.Errorf("identifiers = %v", ids)
	}
	// The classes recorded for Q3500725; at calibration they were Q768186 and Q14748.
	wantTypes := []string{"https://www.wikidata.org/wiki/Q768186", "https://www.wikidata.org/wiki/Q14748"}
	if got := anyStrs(m["additionalType"]); !slices.Equal(got, wantTypes) {
		t.Errorf("additionalType = %v", got)
	}
}
