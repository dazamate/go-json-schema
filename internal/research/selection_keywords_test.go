package research

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestContentWordsDropsStopWords(t *testing.T) {
	for in, want := range map[string][]string{
		"Cost to move a Piano": {"cost", "move", "piano"},
		"movers near me":       {"movers"},
		"IT support":           {"it", "support"},
	} {
		if got := contentWords(in); !slices.Equal(got, want) {
			t.Errorf("contentWords(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKeywordGramsSkipStopWordEdges(t *testing.T) {
	want := []string{"cost", "cost to move", "cost to move a piano", "move", "move a piano", "piano"}
	if got := keywordGrams("cost to move a piano"); !slices.Equal(got, want) {
		t.Errorf("grams = %q, want %q", got, want)
	}
}

func TestKeywordGramsAllRuns(t *testing.T) {
	want := []string{"moving", "moving cost", "moving cost calculator", "cost", "cost calculator", "calculator"}
	if got := keywordGrams("moving cost calculator"); !slices.Equal(got, want) {
		t.Errorf("grams = %q, want %q", got, want)
	}
}

func TestStem(t *testing.T) {
	for in, want := range map[string]string{"moving": "mov", "movers": "mov", "calculator": "calculator", "costs": "cost", "gas": "gas"} {
		if got := stem(in); got != want {
			t.Errorf("stem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSharesStem(t *testing.T) {
	if !sharesStem("Moving company", []string{"moving"}) {
		t.Error("Moving company should share a stem with moving")
	}
	if sharesStem("Truck", []string{"moving", "cost", "calculator"}) {
		t.Error("Truck should share no stem")
	}
}

func TestFocusMatchesStems(t *testing.T) {
	got := focus([]string{"moving", "cost", "calculator"}, "Moving Cost Calculator: Free Estimate", "Movers' price guide")
	want := map[string]FocusWord{"moving": {true, true}, "cost": {true, false}, "calculator": {true, false}}
	for w, f := range want {
		if got[w] != f {
			t.Errorf("focus[%s] = %+v, want %+v", w, got[w], f)
		}
	}
}

func TestSameTitle(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"Moving_company", "moving company", true}, {"ZIP code", "ZIP Code", false}, {"Piano", "piano", true}} {
		if got := sameTitle(c.a, c.b); got != c.want {
			t.Errorf("sameTitle(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

// kwResolver is a resolver on the fake Wikipedia tab, failing the test on
// any search call.
func kwResolver(t *testing.T, w *fakeWiki) *resolver {
	t.Helper()
	p := w.page()
	inner := p.eval
	p.eval = func(current, expr string) (any, error) {
		for _, bad := range []string{"list=search", "srsearch", "wbsearchentities"} {
			if strings.Contains(expr, bad) {
				t.Errorf("search call %q in %.200s", bad, expr)
			}
		}
		return inner(current, expr)
	}
	r := &resolver{ctx: context.Background(), p: p, lang: "en"}
	if err := r.open(); err != nil {
		t.Fatal(err)
	}
	return r
}

var kwKnown = []titleQID{{"K1", "Q101"}, {"K2", "Q102"}}

func TestKeywordLookupsExactAndMissing(t *testing.T) {
	w := &fakeWiki{pages: map[string]wikiPage{"Cost": {qid: "Q1"}}}
	looks, hits, err := kwResolver(t, w).keywordLookups([]string{"cost", "moving cost"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantLooks := []KeywordLookup{{Phrase: "cost", Result: "exact", Title: "Cost"}, {Phrase: "moving cost", Result: "missing"}}
	if !slices.EqualFunc(looks, wantLooks, func(a, b KeywordLookup) bool {
		return a.Phrase == b.Phrase && a.Result == b.Result && a.Title == b.Title && a.Links == 0 && len(a.Chosen) == 0
	}) {
		t.Errorf("lookups = %+v", looks)
	}
	if want := []lookupHit{{"cost", "Cost", "Q1", "exact"}}; !slices.Equal(hits, want) {
		t.Errorf("hits = %+v", hits)
	}
}

func disambigWiki() *fakeWiki {
	return &fakeWiki{pages: map[string]wikiPage{
		"Moving": {qid: "Q9", disambig: true, links: []string{"A", "B", "C"}},
		"A":      {qid: "Q11"}, "B": {qid: "Q12"}, "C": {qid: "Q13"},
	}}
}

func TestKeywordLookupsDisambiguationTieKeepsBoth(t *testing.T) {
	w := disambigWiki()
	w.articleLinks = map[string][]string{"A": {"K1", "K2"}, "B": {"K1"}, "K2": {"B"}, "C": {"Other"}}
	looks, hits, err := kwResolver(t, w).keywordLookups([]string{"moving"}, kwKnown)
	if err != nil {
		t.Fatal(err)
	}
	if len(looks) != 1 || looks[0].Result != "disambiguation" || looks[0].Title != "Moving" || !slices.Equal(looks[0].Chosen, []string{"A", "B"}) || looks[0].Links != 2 {
		t.Errorf("lookups = %+v", looks)
	}
	want := []lookupHit{{"moving", "A", "Q11", "disambiguation"}, {"moving", "B", "Q12", "disambiguation"}}
	if !slices.Equal(hits, want) {
		t.Errorf("hits = %+v", hits)
	}
}

func TestKeywordLookupsDisambiguationWithoutLinksAddsNothing(t *testing.T) {
	looks, hits, err := kwResolver(t, disambigWiki()).keywordLookups([]string{"moving"}, kwKnown)
	if err != nil {
		t.Fatal(err)
	}
	if len(looks) != 1 || looks[0].Result != "disambiguation" || len(looks[0].Chosen) != 0 || len(hits) != 0 {
		t.Errorf("lookups = %+v hits = %+v", looks, hits)
	}
}

func TestKeywordLookupsRedirectNeedsALink(t *testing.T) {
	w := &fakeWiki{
		pages:     map[string]wikiPage{"Moving company": {qid: "Q7"}},
		redirects: map[string][2]string{"Movers": {"Moving company", ""}},
	}
	looks, hits, err := kwResolver(t, w).keywordLookups([]string{"movers"}, kwKnown)
	if err != nil {
		t.Fatal(err)
	}
	if len(looks) != 1 || looks[0].Result != "redirect" || len(hits) != 0 {
		t.Errorf("without a link: lookups = %+v hits = %+v", looks, hits)
	}
	w.articleLinks = map[string][]string{"Moving company": {"K1"}}
	looks, hits, err = kwResolver(t, w).keywordLookups([]string{"movers"}, kwKnown)
	if err != nil {
		t.Fatal(err)
	}
	if want := []lookupHit{{"movers", "Moving company", "Q7", "redirect"}}; !slices.Equal(hits, want) || looks[0].Links != 1 || looks[0].Title != "Moving company" {
		t.Errorf("with a link: lookups = %+v hits = %+v", looks, hits)
	}
}

func TestKeywordLookupsSkipKnownQIDs(t *testing.T) {
	w := &fakeWiki{pages: map[string]wikiPage{"Cost": {qid: "Q1"}}}
	looks, hits, err := kwResolver(t, w).keywordLookups([]string{"cost"}, []titleQID{{"Expense", "Q1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 || len(looks) != 1 || looks[0].Result != "exact" {
		t.Errorf("lookups = %+v hits = %+v", looks, hits)
	}
}

func TestLinkCountsFollowsContinue(t *testing.T) {
	w := &fakeWiki{articleLinks: map[string][]string{"A": {"K1", "K2", "K3"}}, linksPerResponse: 1}
	got, err := kwResolver(t, w).linkCounts([]string{"A"}, []string{"K1", "K2", "K3"})
	if err != nil {
		t.Fatal(err)
	}
	if got["A"] != 3 {
		t.Errorf("counts = %v", got)
	}
}

func TestLinkCountsBatchesFiftyTitles(t *testing.T) {
	w := &fakeWiki{articleLinks: map[string][]string{"K1": {"Z000"}}}
	var options []string
	for i := range 120 {
		o := "Z" + strconv.Itoa(1000 + i)[1:]
		options = append(options, o)
		w.articleLinks[o] = []string{"K1"}
	}
	got, err := kwResolver(t, w).linkCounts(options, []string{"K1"})
	if err != nil {
		t.Fatal(err)
	}
	// Three option batches, each asked forward and reversed.
	if w.fetches != 6 {
		t.Errorf("fetches = %d, want 6", w.fetches)
	}
	for _, o := range options {
		want := 1
		if o == "Z000" {
			want = 2 // K1 also links to it
		}
		if got[o] != want {
			t.Errorf("count[%s] = %d, want %d", o, got[o], want)
		}
	}
}

// bigDisambigWiki has a "Moving" page of 30 options, O00 to O29, where only
// the options at positions 26 and up (O25 to O29) link to a known title.
func bigDisambigWiki() *fakeWiki {
	w := &fakeWiki{pages: map[string]wikiPage{}, articleLinks: map[string][]string{}}
	var opts []string
	for i := range 30 {
		o := "O" + strconv.Itoa(100 + i)[1:]
		opts = append(opts, o)
		w.pages[o] = wikiPage{qid: "Q" + strconv.Itoa(200+i)}
		if i >= 25 {
			w.articleLinks[o] = []string{"K1"}
		}
	}
	w.pages["Moving"] = wikiPage{qid: "Q9", disambig: true, links: opts}
	return w
}

func TestKeywordLookupsCountsEveryDisambiguationOption(t *testing.T) {
	looks, hits, err := kwResolver(t, bigDisambigWiki()).keywordLookups([]string{"moving"}, kwKnown)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"O25", "O26", "O27", "O28", "O29"}
	if len(looks) != 1 || !slices.Equal(looks[0].Chosen, want) || looks[0].Links != 1 || len(hits) != 5 {
		t.Errorf("lookups = %+v hits = %+v", looks, hits)
	}
}

func TestKeywordLookupsFollowsDisambiguationContinue(t *testing.T) {
	w := bigDisambigWiki()
	w.genPerResponse = 7
	looks, hits, err := kwResolver(t, w).keywordLookups([]string{"moving"}, kwKnown)
	if err != nil {
		t.Fatal(err)
	}
	if len(looks) != 1 || len(looks[0].Chosen) != 5 || len(hits) != 5 {
		t.Errorf("lookups = %+v hits = %+v", looks, hits)
	}
}

func TestKeywordLookupsNeverSearch(t *testing.T) {
	// kwResolver fails the test on any search call; a missing phrase would
	// trigger one in resolve.
	w := &fakeWiki{pages: map[string]wikiPage{}}
	if _, _, err := kwResolver(t, w).keywordLookups([]string{"nothing here"}, kwKnown); err != nil {
		t.Fatal(err)
	}
}
