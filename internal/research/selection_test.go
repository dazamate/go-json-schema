package research

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/azurelang"
	"github.com/dazamate/go-json-schema/internal/textrazor"
)

// fakeScorer gives each theme label its keyword and page cosines.
type fakeScorer struct {
	cos    map[string][2]float64
	err    error
	labels []string // every label scored
	pages  []string // every page text scored
}

func (s *fakeScorer) Model() string { return "fake-bge" }

func (s *fakeScorer) Score(_ context.Context, _, page string, labels []string) ([]float64, []float64, error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	s.labels, s.pages = append(s.labels, labels...), append(s.pages, page)
	kw, pg := make([]float64, len(labels)), make([]float64, len(labels))
	for i, l := range labels {
		c, ok := s.cos[l]
		if !ok {
			return nil, nil, fmt.Errorf("no cosines for %q", l)
		}
		kw[i], pg[i] = c[0], c[1]
	}
	return kw, pg, nil
}

// fakeBing links the same entities in any text.
type fakeBing struct {
	calls int
	ents  []azurelang.Entity
}

func (b *fakeBing) Link(context.Context, string, string) (azurelang.Result, error) {
	b.calls++
	return azurelang.Result{Entities: b.ents, Records: 2}, nil
}

// selectPage is a tab where the page read returns raw at the URL the tab
// shows, after the redirects in moved. Every other eval goes to the fake
// Wikipedia, which answers de.wikipedia.org calls as en.wikipedia.org ones.
func selectPage(w *fakeWiki, raw rawPage, moved map[string]string) *fakePage {
	wiki := w.page().eval
	return &fakePage{
		navigate: func(u string) (string, error) {
			if to, ok := moved[u]; ok {
				return to, nil
			}
			return u, nil
		},
		eval: func(current, expr string) (any, error) {
			if expr == pageJS {
				raw.Href = current
				return raw, nil
			}
			return wiki(current, strings.ReplaceAll(expr, "https://de.wikipedia.org", "https://en.wikipedia.org"))
		},
	}
}

const selectURL = "https://a.example/moving-cost-calculator/"

func pageMention(title, qid, freebase, matched string, rel float64) textrazor.Entity {
	return textrazor.Entity{ID: title, WikidataID: qid, FreebaseID: freebase, MatchedText: matched, Relevance: rel, Confidence: 2}
}

// selectFixture is our page and what TextRazor and Wikidata say about it:
//   - A, Moving company: relevance 0.8, topic 1;
//   - B, Truck: 0.6, topic 0.9;
//   - C, Billiard table: 0.1, no topic, with a link;
//   - D, Email: 0.2;
//   - E, Stairs: 0.4, topic 0.1.
type selectFixture struct {
	raw    rawPage
	moved  map[string]string
	ents   []textrazor.Entity
	topics []textrazor.Topic
	wiki   *fakeWiki
}

func newSelectFixture() *selectFixture {
	return &selectFixture{
		raw: rawPage{
			Title: "Moving Cost Calculator", Lang: "en", Headings: []string{"h1: Moving cost calculator", "h2: Pool tables"},
			Text:  "Our moving company calculator prices a truck, stairs and a pool table. Email us.",
			Links: []pageLink{{Href: "https://a.example/pool/", Text: "Moving a pool table"}},
		},
		ents: []textrazor.Entity{
			pageMention("Moving company", "Q1", "/m/a", "moving company", 0.8),
			pageMention("Moving company", "Q1", "/m/a", "Movers", 0.5),
			pageMention("Truck", "Q2", "/m/b", "truck", 0.6),
			pageMention("Billiard table", "Q3", "/m/c", "pool table", 0.1),
			pageMention("Email", "Q4", "", "Email", 0.2),
			pageMention("Stairs", "Q5", "", "stairs", 0.4),
			{MatchedText: "$1,400", Relevance: 0.9}, // no Wikidata ID: dropped without a listing
		},
		topics: []textrazor.Topic{
			{Label: "Moving company", WikidataID: "Q1", Score: 1},
			{Label: "Moving companies", WikidataID: "Q1", Score: 0.1},
			{Label: "Truck", WikidataID: "Q2", Score: 0.9},
			{Label: "Email", WikidataID: "Q4", Score: 0.8},
			{Label: "Stairs", WikidataID: "Q5", Score: 0.1},
		},
		wiki: &fakeWiki{pages: map[string]wikiPage{}, items: map[string]wikiItem{
			"Q1": {sitelink: "Moving company", claims: map[string][]string{"P646": {"/m/a"}, "P31": {"Q100"}}},
			"Q2": {sitelink: "Truck", claims: map[string][]string{"P646": {"/m/b"}}},
			"Q3": {sitelink: "Billiard table"},
			"Q4": {sitelink: "Email"},
			"Q5": {sitelink: "Stairs"},
		}},
	}
}

// researcher returns a Researcher on the fixture's tab and linker, with no
// scorer, Bing or cache.
func (f *selectFixture) researcher() *Researcher {
	return &Researcher{
		Browser: fakeBrowser{selectPage(f.wiki, f.raw, f.moved)},
		Linker: &fakeLinker{link: func(string) (textrazor.Result, error) {
			return textrazor.Result{Entities: f.ents, Topics: f.topics, Units: 1}, nil
		}},
	}
}

func selectQuery() SelectQuery {
	return SelectQuery{URL: selectURL, Keyword: "moving cost calculator", Settings: DefaultSelectSettings}
}

func runSelect(t *testing.T, r *Researcher, q SelectQuery) SelectReport {
	t.Helper()
	rep, err := r.SelectEntities(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func selNames(sel []Selected) []string {
	out := []string{}
	for _, s := range sel {
		out = append(out, s.Name)
	}
	return out
}

func findDropped(rep SelectReport, name string) (Dropped, bool) {
	for _, d := range rep.Dropped {
		if d.Name == name {
			return d, true
		}
	}
	return Dropped{}, false
}

// num reads an optional number, NaN when absent.
func num(p *float64) float64 {
	if p == nil {
		return math.NaN()
	}
	return *p
}

func TestSelectEntitiesGateAndRoles(t *testing.T) {
	f := newSelectFixture()
	rep := runSelect(t, f.researcher(), selectQuery())
	if got := selNames(rep.About); !slices.Equal(got, []string{"Moving company"}) {
		t.Fatalf("about = %v", got)
	}
	a := rep.About[0]
	if a.Source != SourcePage || !slices.Equal(a.QualifiedBy, []string{"scores", "keyword"}) || a.QID != "Q1" ||
		num(a.Relevance) != 0.8 || num(a.Topic) != 1 || a.Mentions != 2 || !slices.Equal(a.Matched, []string{"moving company", "Movers"}) {
		t.Errorf("A = %+v", a)
	}
	if !a.KGMIDChecked || a.KGMID != "/m/a" || a.Wikipedia != "https://en.wikipedia.org/wiki/Moving_company" {
		t.Errorf("A ids = %+v", a)
	}
	if m := thingJSON(t, a.Thing); m["name"] != "Moving company" || !slices.Equal(anyStrs(m["additionalType"]), []string{"https://www.wikidata.org/wiki/Q100"}) {
		t.Errorf("A thing = %s", a.Thing)
	}
	if got := selNames(rep.Mentions); !slices.Equal(got, []string{"Truck", "Billiard table"}) {
		t.Fatalf("mentions = %v", got)
	}
	if b := rep.Mentions[0]; b.Source != SourcePage || !slices.Equal(b.QualifiedBy, []string{"scores"}) {
		t.Errorf("B = %+v", b)
	}
	c := rep.Mentions[1]
	if !slices.Equal(c.QualifiedBy, []string{"link"}) || !slices.Equal(c.LinkedTo, []string{"https://a.example/pool/"}) || num(c.Topic) != 0 {
		t.Errorf("C = %+v", c)
	}
	if b, _ := json.Marshal(c); !strings.Contains(string(b), `"topic":0,`) {
		t.Errorf("C's topic score of 0 is not shown: %s", b)
	}
	for name, reason := range map[string]string{"Email": ReasonBelowRelevance, "Stairs": ReasonBelowTopic} {
		d, ok := findDropped(rep, name)
		if !ok || d.Reason != reason || d.Source != SourcePage {
			t.Errorf("dropped %s = %+v, want %s", name, d, reason)
		}
	}
	if len(rep.Dropped) != 2 {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
	if rep.URL != selectURL || rep.FinalURL != selectURL || rep.Title != "Moving Cost Calculator" || rep.Words != 14 || rep.Keyword != "moving cost calculator" {
		t.Errorf("report = %+v", rep)
	}
	if rep.Settings != DefaultSelectSettings || rep.TextRazor != (TextRazorUsage{Units: 1}) {
		t.Errorf("settings %+v, textrazor %+v", rep.Settings, rep.TextRazor)
	}
}

func TestSelectEntitiesMentionNamedOnlyInTheTitleDropped(t *testing.T) {
	f := newSelectFixture()
	f.raw.Title = "Moving Cost Calculator | Acme Movers"
	f.ents = append(f.ents, pageMention("Acme Corporation", "Q9", "/m/9", "Acme", 0.9))
	f.topics = append(f.topics, textrazor.Topic{Label: "Acme Corporation", WikidataID: "Q9", Score: 1})
	f.wiki.items["Q9"] = wikiItem{sitelink: "Acme Corporation", claims: map[string][]string{"P646": {"/m/9"}}}
	rep := runSelect(t, f.researcher(), selectQuery())
	if got := selNames(rep.Mentions); !slices.Equal(got, []string{"Truck", "Billiard table"}) {
		t.Errorf("mentions = %v", got)
	}
	d, ok := findDropped(rep, "Acme Corporation")
	if !ok || d.Reason != ReasonNotInPageText || d.Source != SourcePage || d.QID != "Q9" || num(d.Relevance) != 0.9 || num(d.Topic) != 1 {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
}

func TestSelectEntitiesLinkToAnotherHostDoesNotCount(t *testing.T) {
	f := newSelectFixture()
	f.raw.Links = []pageLink{{Href: "https://other.example/x", Text: "Moving a pool table"}}
	rep := runSelect(t, f.researcher(), selectQuery())
	if got := selNames(rep.Mentions); !slices.Equal(got, []string{"Truck"}) {
		t.Errorf("mentions = %v", got)
	}
	if d, ok := findDropped(rep, "Billiard table"); !ok || d.Reason != ReasonBelowRelevance {
		t.Errorf("C dropped = %+v", d)
	}
}

func TestSelectEntitiesOwnHostForms(t *testing.T) {
	f := newSelectFixture()
	f.moved = map[string]string{"http://a.example/p": "https://www.a.example/p/"}
	f.raw.Links = []pageLink{
		{Href: "https://www.a.example/p/#top", Text: "Pool table"}, // the page itself
		{Href: "https://a.example/p#calculator", Text: "pool table"},
		{Href: "https://a.example/", Text: "Pool table"}, // the site root
		{Href: "mailto:pool@a.example", Text: "Pool table"},
		{Href: "https://a.example/carpool/", Text: "Carpool tables"}, // not whole words
		{Href: "https://a.example/q/", Text: "Pool table"},
		{Href: "https://a.example/q/", Text: "Our pool table guide"},
	}
	q := selectQuery()
	q.URL = "http://a.example/p"
	rep := runSelect(t, f.researcher(), q)
	if rep.URL != "http://a.example/p" || rep.FinalURL != "https://www.a.example/p/" {
		t.Errorf("url %q, final %q", rep.URL, rep.FinalURL)
	}
	if len(rep.Mentions) != 2 || rep.Mentions[1].Name != "Billiard table" || !slices.Equal(rep.Mentions[1].LinkedTo, []string{"https://a.example/q/"}) {
		t.Fatalf("mentions = %+v", rep.Mentions)
	}
}

// themeFixture adds the themes T1 to T5 (Q11 to Q15, named "Theme 1" to
// "Theme 5" on Wikipedia) with topic scores falling from 0.5.
func themeFixture(f *selectFixture) {
	for i, score := range []float64{0.5, 0.4, 0.3, 0.2, 0.1} {
		qid := fmt.Sprintf("Q1%d", i+1)
		f.topics = append(f.topics, textrazor.Topic{Label: fmt.Sprintf("T%d", i+1), WikidataID: qid, Score: score})
		f.wiki.items[qid] = wikiItem{sitelink: fmt.Sprintf("Theme %d", i+1)}
	}
}

func TestSelectEntitiesThemesNeedBothCutoffs(t *testing.T) {
	f := newSelectFixture()
	themeFixture(f)
	scorer := &fakeScorer{cos: map[string][2]float64{
		"T1": {0.45, 0.60}, "T2": {0.39, 0.60}, "T3": {0.45, 0.54}, "T4": {0.400098, 0.60},
		"T5": {0.3996, 0.60}, // 0.400 rounded, so only the unrounded compare drops it
	}}
	r := f.researcher()
	r.Themes = scorer
	rep := runSelect(t, r, selectQuery())
	if got := selNames(rep.About); !slices.Equal(got, []string{"Moving company", "Theme 1", "Theme 4"}) {
		t.Fatalf("about = %v", got)
	}
	t1, t4 := rep.About[1], rep.About[2]
	if t1.Source != SourceTheme || !slices.Equal(t1.QualifiedBy, []string{"similarity"}) || t1.QID != "Q11" ||
		num(t1.KeywordSimilarity) != 0.45 || num(t1.PageSimilarity) != 0.6 || num(t1.Topic) != 0.5 || t1.KGMIDChecked {
		t.Errorf("T1 = %+v", t1)
	}
	if t1.Relevance != nil || t1.Confidence != nil || t1.Mentions != 0 || len(t1.Matched) != 0 || t1.Matched == nil {
		t.Errorf("T1 has page measurements: %+v", t1)
	}
	if num(t4.KeywordSimilarity) != 0.4 {
		t.Errorf("T4 keyword similarity = %v, want 0.4 (rounded to 3 decimals)", num(t4.KeywordSimilarity))
	}
	for name, want := range map[string][2]float64{"Theme 2": {0.39, 0.6}, "Theme 3": {0.45, 0.54}, "Theme 5": {0.4, 0.6}} {
		d, ok := findDropped(rep, name)
		if !ok || d.Reason != ReasonBelowSimilarity || d.Source != SourceTheme || num(d.KeywordSimilarity) != want[0] || num(d.PageSimilarity) != want[1] {
			t.Errorf("dropped %s = %+v", name, d)
		}
	}
	if got := slices.Sorted(slices.Values(scorer.labels)); !slices.Equal(got, []string{"T1", "T2", "T3", "T4", "T5"}) {
		t.Errorf("scored labels = %v, want only the themes", got)
	}
	if rep.Model != "fake-bge" {
		t.Errorf("model = %q", rep.Model)
	}
}

func TestSelectEntitiesThemeNameIsTheSitelink(t *testing.T) {
	f := newSelectFixture()
	f.topics = append(f.topics, textrazor.Topic{Label: "Office equipment", WikidataID: "Q20", Score: 0.3})
	f.wiki.items["Q20"] = wikiItem{sitelink: "Office supplies"}
	r := f.researcher()
	r.Themes = &fakeScorer{cos: map[string][2]float64{"Office equipment": {0.5, 0.6}}}
	rep := runSelect(t, r, selectQuery())
	if got := selNames(rep.About); !slices.Equal(got, []string{"Moving company", "Office supplies"}) {
		t.Fatalf("about = %v", got)
	}
	if th := rep.About[1]; th.Wikipedia != "https://en.wikipedia.org/wiki/Office_supplies" || thingJSON(t, th.Thing)["name"] != "Office supplies" {
		t.Errorf("theme = %+v %s", th, th.Thing)
	}
}

func TestSelectEntitiesCategoryThemeDropped(t *testing.T) {
	f := newSelectFixture()
	f.topics = append(f.topics, textrazor.Topic{Label: "Computers", WikidataID: "Q21", Score: 0.3})
	f.wiki.items["Q21"] = wikiItem{sitelink: "Category:Computers", claims: map[string][]string{"P31": {"Q4167836"}}}
	scorer := &fakeScorer{cos: map[string][2]float64{"Computers": {0.9, 0.9}}}
	r := f.researcher()
	r.Themes = scorer
	rep := runSelect(t, r, selectQuery())
	if d, ok := findDropped(rep, "Computers"); !ok || d.Reason != ReasonNotAnArticle || d.Source != SourceTheme || d.QID != "Q21" {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
	if slices.Contains(scorer.labels, "Computers") {
		t.Error("a category theme was scored")
	}
}

func TestSelectEntitiesWithoutScorerHasNoThemes(t *testing.T) {
	f := newSelectFixture()
	themeFixture(f)
	rep := runSelect(t, f.researcher(), selectQuery())
	for _, s := range rep.About {
		if s.Source == SourceTheme {
			t.Errorf("theme without a scorer: %+v", s)
		}
	}
	if !strings.Contains(rep.Note, "no themes") || rep.Model != "" {
		t.Errorf("note %q, model %q", rep.Note, rep.Model)
	}
	r := f.researcher()
	r.Themes = &fakeScorer{err: errors.New("model failed to load")}
	rep = runSelect(t, r, selectQuery())
	if got := selNames(rep.About); !slices.Equal(got, []string{"Moving company"}) || !strings.Contains(rep.Note, "no themes: model failed to load") {
		t.Errorf("score error: about %v, note %q", got, rep.Note)
	}
}

func TestSelectEntitiesWithoutLinkerFails(t *testing.T) {
	r := newSelectFixture().researcher()
	r.Linker = nil
	_, err := r.SelectEntities(context.Background(), selectQuery())
	if err == nil || !strings.Contains(err.Error(), "textrazor.api_key") {
		t.Fatalf("err = %v", err)
	}
}

func TestSelectEntitiesRejectsBadQueries(t *testing.T) {
	r := newSelectFixture().researcher()
	for want, q := range map[string]func(*SelectQuery){
		`bad url "ftp://a.example/"`: func(q *SelectQuery) { q.URL = "ftp://a.example/" },
		`bad url "https:///p"`:       func(q *SelectQuery) { q.URL = "https:///p" },
		"no keyword":                 func(q *SelectQuery) { q.Keyword = "  " },
		"min_topic is 2":             func(q *SelectQuery) { q.Settings.MinTopic = 2 },
		"min_page_similarity is -0.1": func(q *SelectQuery) {
			q.Settings.MinPageSimilarity = -0.1
		},
		"min_relevance is NaN": func(q *SelectQuery) { q.Settings.MinRelevance = math.NaN() },
	} {
		query := selectQuery()
		q(&query)
		if _, err := r.SelectEntities(context.Background(), query); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want %q: err = %v", want, err)
		}
	}
	if n := r.Browser.(fakeBrowser).page.visitedMatching("a.example"); n != 0 {
		t.Errorf("a bad query read the page %d times", n)
	}
}

func TestSelectEntitiesFocusNote(t *testing.T) {
	f := newSelectFixture()
	f.raw.Title = "Moving Cost Estimates"
	rep := runSelect(t, f.researcher(), selectQuery())
	if rep.Focus["calculator"] != (FocusWord{InTitle: false, InH1: true}) || rep.Focus["moving"] != (FocusWord{InTitle: true, InH1: true}) {
		t.Errorf("focus = %+v", rep.Focus)
	}
	want := `the title lacks "calculator": the page's themes may describe its own topic rather than the keyword's`
	if !strings.Contains(rep.Note, want) || strings.Contains(rep.Note, "H1 lacks") {
		t.Errorf("note = %q", rep.Note)
	}
}

func TestSelectEntitiesClipsTheTextAndScoresTheSameText(t *testing.T) {
	f := newSelectFixture()
	themeFixture(f)
	f.raw.Text = strings.Repeat("moving company truck ", 200)
	scorer := &fakeScorer{cos: map[string][2]float64{"T1": {0.5, 0.6}, "T2": {0.5, 0.6}, "T3": {0.5, 0.6}, "T4": {0.5, 0.6}, "T5": {0.5, 0.6}}}
	r := f.researcher()
	r.Themes, r.LinkMaxBytes = scorer, 2048
	rep := runSelect(t, r, selectQuery())
	texts := r.Linker.(*fakeLinker).texts
	if len(texts) != 1 || len(texts[0]) > 2048 || !strings.HasPrefix(texts[0], "Moving Cost Calculator\n\nmoving company") {
		t.Fatalf("sent %d texts: %.60q", len(texts), texts)
	}
	if !slices.Equal(scorer.pages, texts) {
		t.Errorf("scored page text differs from the TextRazor text")
	}
	if !strings.Contains(rep.Note, "text cut to 2 KB (textrazor.max_kb_per_page)") {
		t.Errorf("note = %q", rep.Note)
	}
	if rep.Words != 600 {
		t.Errorf("words = %d, want the whole main text's", rep.Words)
	}
}

func TestSelectEntitiesWithoutBingNotes(t *testing.T) {
	rep := runSelect(t, newSelectFixture().researcher(), selectQuery())
	if !strings.Contains(rep.Note, "no Bing IDs: set azure_language.endpoint and azure_language.api_key in the schema-mcp config file") || rep.Azure != nil {
		t.Errorf("note %q, azure %+v", rep.Note, rep.Azure)
	}
	for _, s := range append(rep.About, rep.Mentions...) {
		if s.BingID != "" {
			t.Errorf("%s has bing_id %q", s.Name, s.BingID)
		}
	}
}

func TestSelectEntitiesAttachesCachedBingIDs(t *testing.T) {
	f := newSelectFixture()
	bing := &fakeBing{ents: []azurelang.Entity{{BingID: "b-truck", URL: "https://en.wikipedia.org/wiki/Truck"}}}
	r := f.researcher()
	r.Bing, r.Cache = bing, NewCache(t.TempDir())
	rep := runSelect(t, r, selectQuery())
	if rep.Azure == nil || *rep.Azure != (AzureUsage{Records: 2}) {
		t.Errorf("azure = %+v", rep.Azure)
	}
	if tr := rep.Mentions[0]; tr.Name != "Truck" || tr.BingID != "b-truck" || !strings.Contains(string(tr.Thing), `"BingEntityID"`) {
		t.Errorf("truck = %+v %s", tr, tr.Thing)
	}
	rep = runSelect(t, r, selectQuery())
	if rep.Azure == nil || !rep.Azure.Cached || bing.calls != 1 || rep.Mentions[0].BingID != "b-truck" {
		t.Errorf("second run: azure %+v, calls %d", rep.Azure, bing.calls)
	}
}

// deItem is a Wikidata item with a German Wikipedia sitelink.
func deItem(title, freebase string) json.RawMessage {
	ent := map[string]any{"sitelinks": map[string]any{"dewiki": map[string]any{"site": "dewiki", "title": title}}, "claims": map[string]any{}}
	if freebase != "" {
		ent["claims"] = map[string]any{"P646": []any{map[string]any{"mainsnak": map[string]any{"datavalue": map[string]any{"value": freebase}}, "rank": "normal"}}}
	}
	b, _ := json.Marshal(ent)
	return b
}

func TestSelectEntitiesBingNotForGerman(t *testing.T) {
	f := newSelectFixture()
	f.raw.Lang = "de"
	f.raw.Text = "Die Umzugsfirma fährt den LKW."
	f.ents = []textrazor.Entity{pageMention("Umzugsunternehmen", "Q1", "/m/a", "Umzugsfirma", 0.8), pageMention("Lastkraftwagen", "Q2", "/m/b", "LKW", 0.6)}
	f.topics = []textrazor.Topic{{Label: "Umzug", WikidataID: "Q1", Score: 1}, {Label: "LKW", WikidataID: "Q2", Score: 0.9}}
	f.wiki = &fakeWiki{pages: map[string]wikiPage{}, rawItems: map[string]json.RawMessage{
		"Q1": deItem("Umzugsunternehmen", "/m/a"), "Q2": deItem("Lastkraftwagen", "/m/b"),
	}}
	bing := &fakeBing{}
	r := f.researcher()
	r.Bing = bing
	q := selectQuery()
	q.Keyword = "Umzugskosten Rechner"
	rep := runSelect(t, r, q)
	if rep.Language != "de" || bing.calls != 0 || !strings.Contains(rep.Note, "English and Spanish") || rep.Azure != nil {
		t.Errorf("language %q, bing calls %d, note %q", rep.Language, bing.calls, rep.Note)
	}
	sel := append(rep.About, rep.Mentions...)
	if len(sel) == 0 {
		t.Fatal("nothing selected")
	}
	for _, s := range sel {
		if strings.Contains(strings.ToLower(string(s.Thing)), "dbpedia") || !strings.HasPrefix(s.Wikipedia, "https://de.wikipedia.org/wiki/") ||
			!strings.Contains(string(s.Thing), "https://de.wikipedia.org/wiki/") {
			t.Errorf("%s: wikipedia %q, thing %s", s.Name, s.Wikipedia, s.Thing)
		}
	}
}

func TestSelectEntitiesLanguageFromPage(t *testing.T) {
	for page, want := range map[string]string{"en-US": "en", "": "en", "EN_gb": "en", "x!": "en"} {
		f := newSelectFixture()
		f.raw.Lang = page
		if rep := runSelect(t, f.researcher(), selectQuery()); rep.Language != want {
			t.Errorf("lang %q: language = %q, want %q", page, rep.Language, want)
		}
	}
}

func TestSelectEntitiesLookupHitMovesAPageEntity(t *testing.T) {
	f := newSelectFixture()
	f.ents = append(f.ents, pageMention("Cost", "Q6", "/m/f", "cost", 0.2)) // fails the gate
	f.wiki.pages["Cost"] = wikiPage{qid: "Q6"}
	f.wiki.items["Q6"] = wikiItem{sitelink: "Cost", claims: map[string][]string{"P646": {"/m/f"}}}
	rep := runSelect(t, f.researcher(), selectQuery())
	if got := selNames(rep.About); !slices.Equal(got, []string{"Moving company", "Cost"}) {
		t.Fatalf("about = %v", got)
	}
	c := rep.About[1]
	if c.Source != SourceKeyword || !slices.Equal(c.QualifiedBy, []string{"exact"}) || num(c.Relevance) != 0.2 || !slices.Equal(c.Matched, []string{"cost"}) || c.Mentions != 1 {
		t.Errorf("cost = %+v", c)
	}
	n := 0
	for _, s := range append(rep.About, rep.Mentions...) {
		if s.QID == "Q6" {
			n++
		}
	}
	for _, d := range rep.Dropped {
		if d.QID == "Q6" {
			t.Errorf("selected Q6 is also dropped: %+v", d)
		}
	}
	if n != 1 {
		t.Errorf("Q6 selected %d times", n)
	}
	if !slices.ContainsFunc(rep.KeywordLookups, func(l KeywordLookup) bool { return l.Phrase == "cost" && l.Result == "exact" && l.Title == "Cost" }) {
		t.Errorf("lookups = %+v", rep.KeywordLookups)
	}
}

func TestSelectEntitiesLookupsInKeywordOrder(t *testing.T) {
	f := newSelectFixture()
	f.wiki.pages["Calculator"] = wikiPage{qid: "Q7"}
	f.wiki.pages["Relocation (personal)"] = wikiPage{qid: "Q8"}
	f.wiki.redirects = map[string][2]string{"Moving": {"Relocation (personal)", ""}}
	f.wiki.articleLinks = map[string][]string{"Relocation (personal)": {"Moving company"}}
	f.wiki.items["Q7"] = wikiItem{sitelink: "Calculator", claims: map[string][]string{"P646": {"/m/7"}}}
	f.wiki.items["Q8"] = wikiItem{sitelink: "Relocation (personal)"}
	rep := runSelect(t, f.researcher(), selectQuery())
	// The exact hit (calculator) is found first, but "moving" comes first in the keyword.
	if got := selNames(rep.About); !slices.Equal(got, []string{"Moving company", "Relocation (personal)", "Calculator"}) {
		t.Fatalf("about = %v", got)
	}
	reloc, calc := rep.About[1], rep.About[2]
	if reloc.Source != SourceKeyword || !slices.Equal(reloc.QualifiedBy, []string{"redirect"}) || reloc.Relevance != nil || reloc.KGMIDChecked {
		t.Errorf("relocation = %+v", reloc)
	}
	if !slices.Equal(calc.QualifiedBy, []string{"exact"}) || calc.KGMID != "/m/7" || calc.KGMIDChecked || calc.LinkedTo == nil {
		t.Errorf("calculator = %+v", calc)
	}
}

func TestSelectEntitiesRejectsAPageWithNoText(t *testing.T) {
	f := newSelectFixture()
	f.raw.Text = " \n\t "
	_, err := f.researcher().SelectEntities(context.Background(), selectQuery())
	if err == nil || !strings.Contains(err.Error(), "no main text on "+selectURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestSelectEntitiesRejectsAnErrorPage(t *testing.T) {
	f := newSelectFixture()
	f.raw.Status = 404
	_, err := f.researcher().SelectEntities(context.Background(), selectQuery())
	if err == nil || err.Error() != selectURL+" returned HTTP 404" {
		t.Errorf("status 404: err = %v", err)
	}
	f = newSelectFixture()
	f.moved = map[string]string{selectURL: "chrome-error://chromewebdata/"}
	_, err = f.researcher().SelectEntities(context.Background(), selectQuery())
	if err == nil || err.Error() != selectURL+" did not load (browser ended on chrome-error://chromewebdata/)" {
		t.Errorf("chrome error page: err = %v", err)
	}
}

func TestSelectEntitiesAcceptsAnUnknownOrOKStatus(t *testing.T) {
	for _, status := range []int{0, 200} {
		f := newSelectFixture()
		f.raw.Status = status
		if rep := runSelect(t, f.researcher(), selectQuery()); len(rep.About) == 0 {
			t.Errorf("status %d: nothing selected", status)
		}
	}
}

func TestSelectEntitiesUsesTheTextRazorCache(t *testing.T) {
	f := newSelectFixture()
	r := f.researcher()
	r.Cache = NewCache(t.TempDir())
	if rep := runSelect(t, r, selectQuery()); rep.TextRazor.Cached {
		t.Fatal("first run cached")
	}
	rep := runSelect(t, r, selectQuery())
	if !rep.TextRazor.Cached || rep.TextRazor.Units != 0 {
		t.Errorf("second run textrazor = %+v", rep.TextRazor)
	}
	if n := len(r.Linker.(*fakeLinker).texts); n != 1 {
		t.Errorf("linker called %d times, want 1", n)
	}
	// Our page is never served from the page cache.
	if n := r.Browser.(fakeBrowser).page.visitedMatching("a.example"); n != 2 {
		t.Errorf("page read %d times, want 2", n)
	}
}

func TestSelectReportJSONKeys(t *testing.T) {
	rep := runSelect(t, newSelectFixture().researcher(), selectQuery())
	b, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"url", "final_url", "title", "language", "words", "keyword", "focus", "about", "mentions", "keyword_lookups", "dropped", "settings", "textrazor", "model", "note"} {
		if _, ok := m[k]; !ok {
			t.Errorf("no %q in %s", k, b)
		}
	}
	for _, k := range []string{"keyword_lookups", "dropped", "about", "mentions"} {
		if _, ok := m[k].([]any); !ok {
			t.Errorf("%s is not a list: %v", k, m[k])
		}
	}
	if s := m["settings"].(map[string]any); s["min_relevance"] != 0.3 || s["min_page_similarity"] != 0.55 {
		t.Errorf("settings = %v", s)
	}
}
