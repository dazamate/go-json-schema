package research

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func hoodWiki() *fakeWiki {
	return &fakeWiki{
		pages: map[string]wikiPage{
			"Price action trading": {qid: "Q1"}, "Technical analysis": {qid: "Q2", desc: "Security analysis methodology"},
			"Gap (chart pattern)": {qid: "Q3", desc: "Price chart analysis concept"}, "Action": {qid: "Q8", disambig: true},
			"Orphan": {}, "Day trading": {qid: "Q6"}, "Chart pattern": {qid: "Q7"},
		},
		redirects:        map[string][2]string{"Gap pattern": {"Gap (chart pattern)", ""}},
		articleLinks:     map[string][]string{"Price action trading": {"Technical analysis", "Gap pattern", "Action", "Orphan", "Chart pattern", "Price action trading"}},
		linksHere:        map[string][]string{"Price action trading": {"Day trading", "Chart pattern"}},
		linksPerResponse: 2,
	}
}

func TestGraphHoodLinksBothWays(t *testing.T) {
	h, err := kwResolver(t, hoodWiki()).graphHood("Price action trading")
	if err != nil {
		t.Fatal(err)
	}
	if h.OutLinks != 6 || h.InLinks != 2 || h.InLinksCut {
		t.Errorf("counts out %d in %d cut %v, want 6 2 false", h.OutLinks, h.InLinks, h.InLinksCut)
	}
	want := []neighbour{
		{Title: "Chart pattern", QID: "Q7", Out: true, In: true},
		{Title: "Day trading", QID: "Q6", In: true},
		{Title: "Gap (chart pattern)", QID: "Q3", Desc: "Price chart analysis concept", Out: true},
		{Title: "Technical analysis", QID: "Q2", Desc: "Security analysis methodology", Out: true},
	}
	if !reflect.DeepEqual(h.Neighbours, want) {
		t.Errorf("neighbours = %+v, want %+v", h.Neighbours, want)
	}
}

func TestGraphHoodStopsAtMaxInLinks(t *testing.T) {
	w := &fakeWiki{
		pages:            map[string]wikiPage{"Centre": {qid: "Q1"}},
		linksHere:        map[string][]string{},
		linksPerResponse: 500,
	}
	for i := 0; i <= 500; i++ {
		title := fmt.Sprintf("L%03d", i)
		w.pages[title] = wikiPage{qid: fmt.Sprintf("Q%d", 100+i)}
		w.linksHere["Centre"] = append(w.linksHere["Centre"], title)
	}
	h, err := kwResolver(t, w).graphHood("Centre")
	if err != nil {
		t.Fatal(err)
	}
	if h.InLinks != 500 || !h.InLinksCut || len(h.Neighbours) != 500 {
		t.Errorf("in %d cut %v neighbours %d, want 500 true 500", h.InLinks, h.InLinksCut, len(h.Neighbours))
	}
}

func TestCachedHoodFetchesOnce(t *testing.T) {
	w := hoodWiki()
	wr := kwResolver(t, w)
	r := &Researcher{Cache: NewCache(t.TempDir())}
	first, err := r.cachedHood(wr, "Price action trading")
	if err != nil {
		t.Fatal(err)
	}
	n := w.fetches
	second, err := r.cachedHood(wr, "Price action trading")
	if err != nil {
		t.Fatal(err)
	}
	if w.fetches != n {
		t.Errorf("second call fetched %d times", w.fetches-n)
	}
	if !reflect.DeepEqual(first, second) || len(first.Neighbours) != 4 {
		t.Errorf("hoods differ: %+v vs %+v", first, second)
	}
}

func TestLinkedMembersBothDirections(t *testing.T) {
	w := &fakeWiki{articleLinks: map[string][]string{
		"Foreign exchange market": {"Technical analysis", "Gap (chart pattern)"},
		"Chart pattern":           {"Foreign exchange market"},
		"Candle":                  {"Chart pattern"},
		"Technical analysis":      {"Technical analysis", "Gap (chart pattern)"},
	}}
	got, err := kwResolver(t, w).linkedMembers(
		[]string{"Foreign exchange market", "Candle", "Technical analysis"},
		[]string{"Technical analysis", "Gap (chart pattern)", "Chart pattern"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"Foreign exchange market": {"Chart pattern", "Gap (chart pattern)", "Technical analysis"},
		"Candle":                  {"Chart pattern"},
		"Technical analysis":      {"Gap (chart pattern)"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("linkedMembers = %v, want %v", got, want)
	}
}

func TestLinkedMembersRedirectedTitle(t *testing.T) {
	w := &fakeWiki{
		redirects:    map[string][2]string{"Forex": {"Foreign exchange market", ""}},
		articleLinks: map[string][]string{"Foreign exchange market": {"Technical analysis"}},
	}
	got, err := kwResolver(t, w).linkedMembers([]string{"Forex"}, []string{"Technical analysis"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{"Forex": {"Technical analysis"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("linkedMembers = %v, want %v", got, want)
	}
}

func TestLinkCountsRedirectedKnownTitle(t *testing.T) {
	w := &fakeWiki{
		redirects:    map[string][2]string{"K old": {"K1", ""}},
		articleLinks: map[string][]string{"K1": {"A"}},
	}
	got, err := kwResolver(t, w).linkCounts([]string{"A"}, []string{"K old"})
	if err != nil {
		t.Fatal(err)
	}
	if got["A"] != 1 {
		t.Errorf("counts = %v, want A 1", got)
	}
}

func TestLinkedMembersBatchesAndContinues(t *testing.T) {
	w := &fakeWiki{articleLinks: map[string][]string{}, linksPerResponse: 1}
	var titles []string
	for i := 0; i < 60; i++ {
		title := fmt.Sprintf("T%02d", i)
		titles = append(titles, title)
		w.articleLinks[title] = []string{"M1"}
	}
	got, err := kwResolver(t, w).linkedMembers(titles, []string{"M1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 60 {
		t.Fatalf("%d titles linked, want 60", len(got))
	}
	for _, title := range titles {
		if !reflect.DeepEqual(got[title], []string{"M1"}) {
			t.Errorf("%s = %v, want [M1]", title, got[title])
		}
	}
}

func TestCentreCandidates(t *testing.T) {
	lk := func(phrase, result, title, qid string) KeywordLookup {
		return KeywordLookup{Phrase: phrase, Result: result, Title: title, qid: qid}
	}
	for _, c := range []struct {
		name         string
		looks        []KeywordLookup
		contentWords int
		want         [][]titleQID
	}{
		{"whole phrase", []KeywordLookup{
			lk("price", "exact", "Price", "Q8"), lk("price action", "redirect", "Price action trading", "Q1"),
			lk("price action trading", "exact", "Price action trading", "Q1"), lk("action", "disambiguation", "Action", ""),
			lk("action trading", "missing", "", ""), lk("trading", "redirect", "Trade", "Q5"),
		}, 3, [][]titleQID{{{"Price action trading", "Q1"}}}},
		{"moving", []KeywordLookup{
			lk("moving", "disambiguation", "Moving", ""), lk("moving cost", "missing", "", ""),
			lk("moving cost calculator", "missing", "", ""), lk("cost", "exact", "Cost", "Q2"),
			lk("cost calculator", "missing", "", ""), lk("calculator", "exact", "Calculator", "Q3"),
		}, 3, nil},
		{"one word", []KeywordLookup{lk("plumber", "exact", "Plumber", "Q4")}, 1, [][]titleQID{{{"Plumber", "Q4"}}}},
		{"redirect, no confirmation", []KeywordLookup{
			lk("price", "exact", "Price", "Q8"), lk("price action", "redirect", "Price action trading", "Q1"),
			lk("price action strategy", "missing", "", ""), lk("strategy", "exact", "Strategy", "Q9"),
		}, 3, [][]titleQID{{{"Price action trading", "Q1"}}}},
		{"tie", []KeywordLookup{
			lk("foo bar", "exact", "A", "Q11"), lk("bar baz", "missing", "", ""), lk("baz qux", "exact", "B", "Q12"),
		}, 4, [][]titleQID{{{"A", "Q11"}, {"B", "Q12"}}}},
		{"fallback group", []KeywordLookup{
			lk("x y z", "exact", "XYZ", "Q21"), lk("x y", "exact", "XY", "Q22"), lk("y z", "redirect", "XYZ", "Q21"),
		}, 3, [][]titleQID{{{"XYZ", "Q21"}}, {{"XY", "Q22"}}}},
		{"stop words", []KeywordLookup{
			lk("trade", "exact", "Trade", "Q5"), lk("trade price", "missing", "", ""), lk("trade price action", "missing", "", ""),
			lk("price", "exact", "Price", "Q8"), lk("price action", "redirect", "Price action trading", "Q1"),
			lk("action", "disambiguation", "Action", ""),
		}, 3, [][]titleQID{{{"Price action trading", "Q1"}}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := centreCandidates(c.looks, c.contentWords); !reflect.DeepEqual(got, c.want) {
				t.Errorf("centreCandidates = %v, want %v", got, c.want)
			}
		})
	}
}

func TestKeywordLookupsKeepCentreQIDs(t *testing.T) {
	w := &fakeWiki{
		pages:     map[string]wikiPage{"Price action trading": {qid: "Q1"}},
		redirects: map[string][2]string{"Price action": {"Price action trading", ""}},
	}
	looks, hits, err := kwResolver(t, w).keywordLookups([]string{"price action trading", "price action"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(looks) != 2 || looks[0].Result != "exact" || looks[1].Result != "redirect" || looks[0].qid != "Q1" || looks[1].qid != "Q1" {
		t.Errorf("lookups = %+v", looks)
	}
	if want := []lookupHit{{"price action trading", "Price action trading", "Q1", "exact"}}; !slices.Equal(hits, want) {
		t.Errorf("hits = %+v", hits)
	}
}

func TestNamedForms(t *testing.T) {
	for _, c := range []struct {
		title, text string
		want        []string
	}{
		{"Island reversal", "Two island reversals formed.", []string{"island reversals"}},
		{"Market reversal", "market reversals and a Market Reversal", []string{"Market Reversal", "market reversals"}},
		{"Head and shoulders (chart pattern)", "a head and shoulders top", []string{"head and shoulders"}},
		{"Bull market", "A bull marketing push", nil},
		{"Gap (chart pattern)", "Trade the Gap today.", nil}, // one word once the bracket goes
		{"Doji", "doji and DOJI", nil},                       // one-word titles are named only by TextRazor
	} {
		if got := namedForms(c.title, c.text); !slices.Equal(got, c.want) {
			t.Errorf("namedForms(%q, %q) = %q, want %q", c.title, c.text, got, c.want)
		}
	}
}

func pruneNs() []neighbour {
	return []neighbour{{Title: "A"}, {Title: "B"}, {Title: "C"}, {Title: "D"}}
}

func keptTitles(p pruned) []string {
	var out []string
	for _, k := range p.Kept {
		out = append(out, k.Title)
	}
	return out
}

func TestPruneKeepsNamedAndCuts(t *testing.T) {
	sims := map[string]float64{"A": .6, "B": .39, "C": .30, "D": .45}
	p := pruneNeighbours(pruneNs(), sims, map[string][]string{"B": {"b"}}, DefaultSelectSettings)
	if got := keptTitles(p); !slices.Equal(got, []string{"A", "D", "B"}) {
		t.Errorf("kept = %v", got)
	}
	want := []Dropped{{Name: "C", Source: SourceGraph, GraphSimilarity: f64(0.3), Reason: ReasonBelowGraphSimilarity}}
	if !reflect.DeepEqual(p.Dropped, want) {
		t.Errorf("dropped = %+v", p.Dropped)
	}
	if p.Cut != 1 || p.Capped != 0 {
		t.Errorf("cut %d capped %d", p.Cut, p.Capped)
	}
	if p.Kept[2].Sim == nil || *p.Kept[2].Sim != .39 || !slices.Equal(p.Kept[2].Named, []string{"b"}) || p.Kept[0].Named != nil {
		t.Errorf("kept = %+v", p.Kept)
	}
}

func TestPruneCapsUnnamed(t *testing.T) {
	set := DefaultSelectSettings
	set.MaxGraphNeighbours = 2
	sims := map[string]float64{"A": .6, "D": .45, "E": .5, "B": .2}
	ns := append(pruneNs(), neighbour{Title: "E"})
	p := pruneNeighbours(ns, sims, map[string][]string{"B": {"b"}}, set)
	if got := keptTitles(p); !slices.Equal(got, []string{"A", "E", "B"}) {
		t.Errorf("kept = %v", got)
	}
	if p.Capped != 1 || len(p.Dropped) != 0 {
		t.Errorf("capped %d dropped %+v", p.Capped, p.Dropped)
	}
}

func TestPruneWithoutModel(t *testing.T) {
	p := pruneNeighbours(pruneNs(), nil, map[string][]string{"B": {"b"}}, DefaultSelectSettings)
	if got := keptTitles(p); !slices.Equal(got, []string{"B"}) {
		t.Errorf("kept = %v", got)
	}
	if len(p.Dropped) != 0 || p.Cut != 3 || p.Capped != 0 {
		t.Errorf("dropped %+v cut %d capped %d", p.Dropped, p.Cut, p.Capped)
	}
}

func TestPruneZeroCap(t *testing.T) {
	set := DefaultSelectSettings
	set.MaxGraphNeighbours = 0
	sims := map[string]float64{"A": .6, "B": .2, "C": .3, "D": .45}
	p := pruneNeighbours(pruneNs(), sims, map[string][]string{"B": {"b"}}, set)
	if got := keptTitles(p); !slices.Equal(got, []string{"B"}) {
		t.Errorf("kept = %v", got)
	}
	if p.Capped != 2 {
		t.Errorf("capped %d", p.Capped)
	}
}

func TestGraphText(t *testing.T) {
	if got := graphText(neighbour{Title: "Gap", Desc: "A concept"}); got != "Gap: A concept" {
		t.Errorf("got %q", got)
	}
	if got := graphText(neighbour{Title: "Gap"}); got != "Gap" {
		t.Errorf("got %q", got)
	}
}
