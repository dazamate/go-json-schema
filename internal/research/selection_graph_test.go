package research

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/textrazor"
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

const (
	patKeyword = "price action trading"
	patURL     = "https://forex.example/price-action-trading/"
	patText    = "Price action trading reads every candle. Technical analysis and forex go together. Island reversals are rare, so trade the gap."
)

func patWiki() *fakeWiki {
	item := func(title string, claims map[string][]string) wikiItem {
		return wikiItem{sitelink: title, claims: claims}
	}
	return &fakeWiki{
		pages: map[string]wikiPage{
			"Price action trading": {qid: "Q1"}, "Technical analysis": {qid: "Q2", desc: "Security analysis methodology"},
			"Gap (chart pattern)": {qid: "Q3", desc: "Price chart analysis concept"}, "ISBN": {qid: "Q4", desc: "Unique numeric book identifier"},
			"Island reversal": {qid: "Q5"}, "Day trading": {qid: "Q6", desc: "Buying and selling within the same day"},
			"Chart pattern": {qid: "Q7", desc: "Pattern within a chart"}, "Action": {qid: "Q8", disambig: true}, "Orphan": {},
			"Foreign exchange market": {qid: "Q10"}, "Candle": {qid: "Q11"}, "Trade": {qid: "Q12"}, "Swing trading": {qid: "Q13"},
			"Microeconomics": {qid: "Q14"},
		},
		redirects: map[string][2]string{"Price action": {"Price action trading", ""}},
		items: map[string]wikiItem{
			"Q1": item("Price action trading", nil), "Q2": item("Technical analysis", nil), "Q3": item("Gap (chart pattern)", nil),
			"Q4": item("ISBN", nil), "Q5": item("Island reversal", nil), "Q6": item("Day trading", nil), "Q7": item("Chart pattern", nil),
			"Q10": item("Foreign exchange market", nil), "Q11": item("Candle", nil), "Q12": item("Trade", nil),
			"Q13": item("Swing trading", nil), "Q14": item("Microeconomics", nil),
		},
		articleLinks: map[string][]string{
			"Price action trading":    {"Technical analysis", "Gap (chart pattern)", "ISBN", "Island reversal", "Action", "Orphan"},
			"Day trading":             {"Price action trading"},
			"Chart pattern":           {"Price action trading", "Foreign exchange market"},
			"Foreign exchange market": {"Technical analysis", "Gap (chart pattern)"},
			"Candle":                  {"Chart pattern"},
			"Trade":                   {"Technical analysis", "Day trading"},
			"Swing trading":           {"Technical analysis", "Gap (chart pattern)", "Chart pattern"},
			"Technical analysis":      {"Gap (chart pattern)"},
		},
		linksHere: map[string][]string{"Price action trading": {"Day trading", "Chart pattern"}},
	}
}

func patRaw() rawPage {
	return rawPage{Title: "Price Action Trading", Lang: "en", Headings: []string{"h1: Price action trading"}, Text: patText}
}

func patEntities() []textrazor.Entity {
	return []textrazor.Entity{
		pageMention("Technical analysis", "Q2", "", "technical analysis", 0.5),
		pageMention("Foreign exchange market", "Q10", "", "forex", 0.7),
		pageMention("Candle", "Q11", "", "candle", 0.58),
		pageMention("Trade", "Q12", "", "trade", 0.87),
		pageMention("Swing trading", "Q13", "", "swing trading", 0.12),
	}
}

func patTopics() []textrazor.Topic {
	return []textrazor.Topic{
		{Label: "Technical analysis", WikidataID: "Q2", Score: 0.9}, {Label: "Foreign exchange market", WikidataID: "Q10", Score: 1},
		{Label: "Candle", WikidataID: "Q11", Score: 1}, {Label: "Trade", WikidataID: "Q12", Score: 1},
		{Label: "Microeconomics", WikidataID: "Q14", Score: 0.5},
	}
}

func patSims() map[string]float64 {
	return map[string]float64{
		"Technical analysis: Security analysis methodology": 0.42, "Gap (chart pattern): Price chart analysis concept": 0.548,
		"ISBN: Unique numeric book identifier": 0.299, "Island reversal": 0.325,
		"Day trading: Buying and selling within the same day": 0.576, "Chart pattern: Pattern within a chart": 0.542,
	}
}

var patGroups = [][]titleQID{{{"Price action trading", "Q1"}}}

// patGraph builds the pat graph over w with text as the page's main text.
func patGraph(t *testing.T, w *fakeWiki, themes ThemeScorer, text string, ents []textrazor.Entity, groups [][]titleQID) (*builtGraph, []string) {
	t.Helper()
	r := &Researcher{Cache: NewCache(t.TempDir()), Themes: themes}
	s := newSelection(textrazor.Result{Entities: ents, Topics: patTopics()}, text, nil, patURL, wordsOf(patKeyword), DefaultSelectSettings)
	g, notes, err := r.buildGraph(context.Background(), kwResolver(t, w), patKeyword, s, nil, groups, DefaultSelectSettings)
	if err != nil {
		t.Fatal(err)
	}
	return g, notes
}

func memberTitles(g *builtGraph) []string {
	var out []string
	for _, m := range g.Members {
		out = append(out, m.Title)
	}
	return out
}

func hasNote(notes []string, sub string) bool {
	return slices.ContainsFunc(notes, func(n string) bool { return strings.Contains(n, sub) })
}

func TestBuildGraphKeepsNamedAndScored(t *testing.T) {
	g, notes := patGraph(t, patWiki(), &fakeScorer{sim: patSims()}, patText, patEntities(), patGroups)
	if g == nil {
		t.Fatalf("nil graph, notes %v", notes)
	}
	want := []string{"Price action trading", "Day trading", "Gap (chart pattern)", "Chart pattern", "Technical analysis", "Island reversal"}
	if got := memberTitles(g); !slices.Equal(got, want) {
		t.Errorf("members = %v, want %v", got, want)
	}
	named := map[string][]string{}
	for _, m := range g.Members {
		named[m.Title] = m.Named
	}
	if !g.Members[0].Centre || named["Gap (chart pattern)"] != nil ||
		!slices.Equal(named["Island reversal"], []string{"Island reversals"}) || !slices.Equal(named["Technical analysis"], []string{"technical analysis"}) {
		t.Errorf("centre %v named %v", g.Members[0].Centre, named)
	}
	wantRep := GraphReport{Centres: []string{"Price action trading"}, OutLinks: 6, InLinks: 2, Neighbours: 6, Kept: 5, Cut: 1}
	if !reflect.DeepEqual(g.Report, wantRep) || !g.Scored {
		t.Errorf("report = %+v scored %v, want %+v", g.Report, g.Scored, wantRep)
	}
	wantDropped := []Dropped{{Name: "ISBN", QID: "Q4", Source: SourceGraph, GraphSimilarity: f64(0.299), Reason: ReasonBelowGraphSimilarity}}
	if !reflect.DeepEqual(g.Dropped, wantDropped) {
		t.Errorf("dropped = %+v", g.Dropped)
	}
	for title, want := range map[string][]string{
		"Foreign exchange market": {"Chart pattern", "Gap (chart pattern)", "Technical analysis"},
		"Candle":                  {"Chart pattern"},
		"Trade":                   {"Day trading", "Technical analysis"},
		"Swing trading":           {"Chart pattern", "Gap (chart pattern)", "Technical analysis"},
	} {
		if !slices.Equal(g.Links[title], want) {
			t.Errorf("links[%s] = %v, want %v", title, g.Links[title], want)
		}
	}
	if !hasNote(notes, "graph of Price action trading: 1 cut below min_graph_similarity, 0 over max_graph_neighbours") {
		t.Errorf("notes = %v", notes)
	}
}

func TestBuildGraphDropsAListNeighbour(t *testing.T) {
	w := patWiki()
	w.pages["List of chart patterns"] = wikiPage{qid: "Q15"}
	w.items["Q15"] = wikiItem{sitelink: "List of chart patterns", claims: map[string][]string{"P31": {"Q13406463"}}}
	w.articleLinks["Price action trading"] = append(w.articleLinks["Price action trading"], "List of chart patterns")
	sims := patSims()
	sims["List of chart patterns"] = 0.6
	g, notes := patGraph(t, w, &fakeScorer{sim: sims}, patText, patEntities(), patGroups)
	if g == nil {
		t.Fatalf("nil graph, notes %v", notes)
	}
	var found *Dropped
	for i, d := range g.Dropped {
		if d.Name == "List of chart patterns" {
			found = &g.Dropped[i]
		}
	}
	if found == nil || found.Reason != ReasonNotAnArticle || found.Source != SourceGraph || found.QID != "Q15" {
		t.Errorf("dropped = %+v", g.Dropped)
	}
	if slices.Contains(memberTitles(g), "List of chart patterns") || g.Report.Kept != 5 {
		t.Errorf("members %v kept %d", memberTitles(g), g.Report.Kept)
	}
}

func TestBuildGraphTriesTheNextCentre(t *testing.T) {
	w := patWiki()
	w.pages["XY"] = wikiPage{qid: "Q22"}
	w.items["Q21"] = wikiItem{sitelink: "Category:XYZ"}
	w.items["Q22"] = wikiItem{sitelink: "XY"}
	w.articleLinks["XY"] = []string{"Technical analysis", "Gap (chart pattern)", "Day trading"}
	groups := [][]titleQID{{{"XYZ", "Q21"}}, {{"XY", "Q22"}}}
	g, notes := patGraph(t, w, &fakeScorer{sim: patSims()}, patText, patEntities(), groups)
	if g == nil {
		t.Fatalf("nil graph, notes %v", notes)
	}
	if want := []string{"XY", "Day trading", "Gap (chart pattern)", "Technical analysis"}; !slices.Equal(memberTitles(g), want) {
		t.Errorf("members = %v, want %v", memberTitles(g), want)
	}
	if !slices.Equal(g.Report.Centres, []string{"XY"}) {
		t.Errorf("centres = %v", g.Report.Centres)
	}
}

func TestBuildGraphNoCentre(t *testing.T) {
	w := patWiki()
	w.items["Q21"] = wikiItem{sitelink: "Category:XYZ"}
	g, notes := patGraph(t, w, &fakeScorer{sim: patSims()}, patText, patEntities(), [][]titleQID{{{"XYZ", "Q21"}}})
	if g != nil || !hasNote(notes, "no centre: XYZ failed its identity check (identity_mismatch): about uses the page's themes") {
		t.Errorf("graph %v notes %v", g, notes)
	}
}

func TestBuildGraphTooFewNeighbours(t *testing.T) {
	sims := map[string]float64{}
	for k := range patSims() {
		sims[k] = 0.1
	}
	g, notes := patGraph(t, patWiki(), &fakeScorer{sim: sims}, "Candles form over time.", nil, patGroups)
	if g != nil || !hasNote(notes, "Price action trading has 0 kept neighbours (fewer than 3)") {
		t.Errorf("graph %v notes %v", g, notes)
	}
}

func TestBuildGraphWithoutModel(t *testing.T) {
	g, notes := patGraph(t, patWiki(), nil, patText+" A chart pattern helps.", patEntities(), patGroups)
	if g == nil {
		t.Fatalf("nil graph, notes %v", notes)
	}
	if want := []string{"Price action trading", "Chart pattern", "Island reversal", "Technical analysis"}; !slices.Equal(memberTitles(g), want) {
		t.Errorf("members = %v, want %v", memberTitles(g), want)
	}
	if g.Report.Cut != 3 || g.Scored {
		t.Errorf("cut %d scored %v", g.Report.Cut, g.Scored)
	}
	for _, d := range g.Dropped {
		if d.Name == "ISBN" {
			t.Errorf("ISBN dropped: %+v", d)
		}
	}
	if !hasNote(notes, "graph not scored: no embedding model is set up") {
		t.Errorf("notes = %v", notes)
	}
}

// shortScorer returns one similarity fewer than the texts it is given.
type shortScorer struct{ fakeScorer }

func (s *shortScorer) Similarity(ctx context.Context, keyword string, texts []string) ([]float64, error) {
	got, err := s.fakeScorer.Similarity(ctx, keyword, texts)
	if err != nil {
		return nil, err
	}
	return got[:len(got)-1], nil
}

func TestBuildGraphShortScorerResult(t *testing.T) {
	g, notes := patGraph(t, patWiki(), &shortScorer{fakeScorer{sim: patSims()}}, patText+" A chart pattern helps.", patEntities(), patGroups)
	if g == nil {
		t.Fatalf("nil graph, notes %v", notes)
	}
	if !hasNote(notes, "graph not scored: 5 similarities for 6 neighbours: only the neighbours the page names are kept") || g.Scored {
		t.Errorf("scored %v notes %v", g.Scored, notes)
	}
}

func TestBuildGraphScoringError(t *testing.T) {
	g, notes := patGraph(t, patWiki(), &fakeScorer{err: fmt.Errorf("model down")}, patText+" A chart pattern helps.", patEntities(), patGroups)
	if g == nil {
		t.Fatalf("nil graph, notes %v", notes)
	}
	if !hasNote(notes, "graph not scored: model down: only the neighbours the page names are kept") || g.Scored {
		t.Errorf("scored %v notes %v", g.Scored, notes)
	}
}

func TestBuildGraphReportsScoringProgress(t *testing.T) {
	var msgs []string
	ctx := WithProgress(context.Background(), func(msg string) { msgs = append(msgs, msg) })
	r := &Researcher{Cache: NewCache(t.TempDir()), Themes: &fakeScorer{sim: patSims()}}
	s := newSelection(textrazor.Result{Entities: patEntities(), Topics: patTopics()}, patText, nil, patURL, wordsOf(patKeyword), DefaultSelectSettings)
	if _, _, err := r.buildGraph(ctx, kwResolver(t, patWiki()), patKeyword, s, nil, patGroups, DefaultSelectSettings); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(msgs, "scoring 6 neighbours against the keyword") {
		t.Errorf("progress = %v", msgs)
	}
}

// patResearcher is a Researcher on the pat page, linked as ents and topics,
// scored by scorer.
func patResearcher(t *testing.T, raw rawPage, ents []textrazor.Entity, topics []textrazor.Topic, scorer ThemeScorer) *Researcher {
	t.Helper()
	return patResearcherWiki(t, patWiki(), raw, ents, topics, scorer)
}

func patResearcherWiki(t *testing.T, w *fakeWiki, raw rawPage, ents []textrazor.Entity, topics []textrazor.Topic, scorer ThemeScorer) *Researcher {
	t.Helper()
	return &Researcher{
		Browser: fakeBrowser{selectPage(w, raw, nil)},
		Linker: &fakeLinker{link: func(string) (textrazor.Result, error) {
			return textrazor.Result{Entities: ents, Topics: topics, Units: 1}, nil
		}},
		Themes: scorer,
		Cache:  NewCache(t.TempDir()),
	}
}

func patQuery(set SelectSettings) SelectQuery {
	return SelectQuery{URL: patURL, Keyword: patKeyword, Settings: set}
}

func TestSelectEntitiesGraphRoles(t *testing.T) {
	scorer := &fakeScorer{sim: patSims()}
	rep := runSelect(t, patResearcher(t, patRaw(), patEntities(), patTopics(), scorer), patQuery(DefaultSelectSettings))
	want := []string{"Price action trading", "Technical analysis", "Foreign exchange market", "Day trading", "Gap (chart pattern)", "Chart pattern", "Island reversal"}
	if got := selNames(rep.About); !slices.Equal(got, want) {
		t.Fatalf("about = %v, want %v", got, want)
	}
	by := map[string]Selected{}
	for _, s := range rep.About {
		by[s.Name] = s
	}
	centre := by["Price action trading"]
	if b, _ := json.Marshal(centre); centre.Source != SourceKeyword || !slices.Equal(centre.QualifiedBy, []string{"centre"}) || strings.Contains(string(b), "graph_links") {
		t.Errorf("centre = %s", b)
	}
	ta := by["Technical analysis"]
	if ta.Source != SourcePage || !slices.Equal(ta.QualifiedBy, []string{"out_link", "named"}) || ta.GraphLinks != 2 {
		t.Errorf("Technical analysis = %+v", ta)
	}
	fx := by["Foreign exchange market"]
	if fx.Source != SourcePage || !slices.Equal(fx.QualifiedBy, []string{"scores", "graph_links"}) || fx.GraphLinks != 3 {
		t.Errorf("Foreign exchange market = %+v", fx)
	}
	day := by["Day trading"]
	if day.Source != SourceGraph || !slices.Equal(day.QualifiedBy, []string{"in_link"}) || num(day.GraphSimilarity) != 0.576 || day.Matched == nil || len(day.Matched) != 0 || day.LinkedTo == nil {
		t.Errorf("Day trading = %+v", day)
	}
	gap := by["Gap (chart pattern)"]
	if gap.Source != SourceGraph || !slices.Equal(gap.QualifiedBy, []string{"out_link"}) || len(gap.Matched) != 0 {
		t.Errorf("Gap = %+v", gap)
	}
	if _, ok := thingJSON(t, gap.Thing)["alternateName"]; ok {
		t.Errorf("Gap thing = %s", gap.Thing)
	}
	island := by["Island reversal"]
	if !slices.Equal(island.QualifiedBy, []string{"out_link", "named"}) || num(island.GraphSimilarity) != 0.325 || !slices.Equal(island.Matched, []string{"Island reversals"}) {
		t.Errorf("Island reversal = %+v", island)
	}
	if got := anyStrs(thingJSON(t, island.Thing)["alternateName"]); !slices.Equal(got, []string{"Island reversals"}) {
		t.Errorf("Island reversal alternateName = %v", got)
	}
	if rep.Model != "fake-bge" {
		t.Errorf("model = %q", rep.Model)
	}
	if got := selNames(rep.Mentions); !slices.Equal(got, []string{"Trade"}) {
		t.Fatalf("mentions = %v", got)
	}
	if m := rep.Mentions[0]; !slices.Equal(m.QualifiedBy, []string{"scores", "graph_links"}) || m.GraphLinks != 2 {
		t.Errorf("Trade = %+v", m)
	}
	for name, want := range map[string]Dropped{
		"ISBN":           {Reason: ReasonBelowGraphSimilarity, Source: SourceGraph},
		"Candle":         {Reason: ReasonOffGraph, Source: SourcePage, GraphLinks: new(1)},
		"Swing trading":  {Reason: ReasonOffGraph, Source: SourcePage, GraphLinks: new(3)},
		"Microeconomics": {Reason: ReasonOffGraph, Source: SourceTheme},
	} {
		d, ok := findDropped(rep, name)
		if !ok || d.Reason != want.Reason || d.Source != want.Source || (want.GraphLinks != nil && (d.GraphLinks == nil || *d.GraphLinks != *want.GraphLinks)) {
			t.Errorf("dropped %s = %+v, want %+v", name, d, want)
		}
	}
	if d, _ := findDropped(rep, "Microeconomics"); num(d.Topic) != 0.5 {
		t.Errorf("Microeconomics topic = %v", d.Topic)
	}
	if n := len(rep.Dropped); n != 4 {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
	wantRep := &GraphReport{Centres: []string{"Price action trading"}, OutLinks: 6, InLinks: 2, Neighbours: 6, Kept: 5, Cut: 1}
	if !reflect.DeepEqual(rep.Graph, wantRep) {
		t.Errorf("graph = %+v, want %+v", rep.Graph, wantRep)
	}
	if len(scorer.labels) != 0 {
		t.Errorf("themes scored: %v", scorer.labels)
	}
}

func TestSelectEntitiesGraphUngatedThreshold(t *testing.T) {
	set := DefaultSelectSettings
	set.MinGraphLinksUngated = 3
	rep := runSelect(t, patResearcher(t, patRaw(), patEntities(), patTopics(), &fakeScorer{sim: patSims()}), patQuery(set))
	want := []string{"Price action trading", "Technical analysis", "Foreign exchange market", "Swing trading", "Day trading", "Gap (chart pattern)", "Chart pattern", "Island reversal"}
	if got := selNames(rep.About); !slices.Equal(got, want) {
		t.Fatalf("about = %v, want %v", got, want)
	}
	if s := rep.About[3]; !slices.Equal(s.QualifiedBy, []string{"graph_links"}) || s.GraphLinks != 3 || s.Source != SourcePage {
		t.Errorf("Swing trading = %+v", s)
	}
}

func TestSelectEntitiesGraphCap(t *testing.T) {
	set := DefaultSelectSettings
	set.MaxGraphNeighbours = 2
	rep := runSelect(t, patResearcher(t, patRaw(), patEntities(), patTopics(), &fakeScorer{sim: patSims()}), patQuery(set))
	if got := selNames(rep.About); slices.Contains(got, "Chart pattern") || !slices.Contains(got, "Day trading") || !slices.Contains(got, "Gap (chart pattern)") {
		t.Errorf("about = %v", got)
	}
	if _, ok := findDropped(rep, "Chart pattern"); ok {
		t.Errorf("Chart pattern dropped: %+v", rep.Dropped)
	}
	if rep.Graph == nil || rep.Graph.Capped != 1 {
		t.Errorf("graph = %+v", rep.Graph)
	}
}

func TestSelectEntitiesCentreLinkedOnThePage(t *testing.T) {
	ents := append(patEntities(), pageMention("Price action trading", "Q1", "", "price action trading", 0.9))
	topics := append(patTopics(), textrazor.Topic{Label: "Price action trading", WikidataID: "Q1", Score: 1})
	rep := runSelect(t, patResearcher(t, patRaw(), ents, topics, &fakeScorer{sim: patSims()}), patQuery(DefaultSelectSettings))
	n := 0
	for _, s := range slices.Concat(rep.About, rep.Mentions) {
		if s.Name == "Price action trading" {
			n++
		}
	}
	if n != 1 || len(rep.About) == 0 {
		t.Fatalf("Price action trading listed %d times: about %v mentions %v", n, selNames(rep.About), selNames(rep.Mentions))
	}
	c := rep.About[0]
	if c.Name != "Price action trading" || c.Source != SourceKeyword || !slices.Contains(c.QualifiedBy, "centre") || num(c.Relevance) != 0.9 {
		t.Errorf("centre = %+v", c)
	}
	if _, ok := findDropped(rep, "Price action trading"); ok {
		t.Errorf("centre dropped: %+v", rep.Dropped)
	}
	// TextRazor's own qualifiers first, then centre
	if want := []string{"scores", "centre"}; !slices.Equal(c.QualifiedBy, want) {
		t.Errorf("centre qualified_by = %v, want %v", c.QualifiedBy, want)
	}
}

func TestSelectEntitiesListNeighbourDroppedOnce(t *testing.T) {
	w := patWiki()
	w.pages["List of chart patterns"] = wikiPage{qid: "Q15"}
	w.items["Q15"] = wikiItem{sitelink: "List of chart patterns", claims: map[string][]string{"P31": {"Q13406463"}}}
	w.articleLinks["Price action trading"] = append(w.articleLinks["Price action trading"], "List of chart patterns")
	w.articleLinks["List of chart patterns"] = []string{"Technical analysis", "Gap (chart pattern)", "Chart pattern", "Day trading"}
	sims := patSims()
	sims["List of chart patterns"] = 0.6
	ents := append(patEntities(), pageMention("List of chart patterns", "Q15", "", "list of chart patterns", 0.9))
	rep := runSelect(t, patResearcherWiki(t, w, patRaw(), ents, patTopics(), &fakeScorer{sim: sims}), patQuery(DefaultSelectSettings))
	n := 0
	for _, d := range rep.Dropped {
		if d.Name == "List of chart patterns" {
			n++
			if d.Reason != ReasonNotAnArticle {
				t.Errorf("dropped = %+v", d)
			}
		}
	}
	if n != 1 {
		t.Errorf("dropped %d times: %+v", n, rep.Dropped)
	}
	if slices.Contains(selNames(rep.About), "List of chart patterns") || slices.Contains(selNames(rep.Mentions), "List of chart patterns") {
		t.Errorf("selected: about %v mentions %v", selNames(rep.About), selNames(rep.Mentions))
	}
}

func TestSelectEntitiesGraphLinksUsesTextRazorTitle(t *testing.T) {
	ents := patEntities()
	ents[0] = pageMention("Technical Analysis", "Q2", "", "technical analysis", 0.5)
	w := patWiki()
	w.redirects["Technical Analysis"] = [2]string{"Technical analysis", ""}
	w.articleLinks["Price action trading"] = append(w.articleLinks["Price action trading"], "Technical Analysis")
	rep := runSelect(t, patResearcherWiki(t, w, patRaw(), ents, patTopics(), &fakeScorer{sim: patSims()}), patQuery(DefaultSelectSettings))
	for _, s := range rep.About {
		if s.QID == "Q2" {
			if s.GraphLinks != 2 {
				t.Errorf("Technical Analysis graph_links = %d, want 2", s.GraphLinks)
			}
			return
		}
	}
	t.Errorf("Q2 not in about: %v", selNames(rep.About))
}

func TestSelectEntitiesGraphMentionNotInPageText(t *testing.T) {
	ents := patEntities()
	ents[3] = pageMention("Trade", "Q12", "", "tradez", 0.87)
	rep := runSelect(t, patResearcher(t, patRaw(), ents, patTopics(), &fakeScorer{sim: patSims()}), patQuery(DefaultSelectSettings))
	if slices.Contains(selNames(rep.Mentions), "Trade") {
		t.Errorf("mentions = %v", selNames(rep.Mentions))
	}
	if d, ok := findDropped(rep, "Trade"); !ok || d.Reason != ReasonNotInPageText {
		t.Errorf("dropped Trade = %+v, %v", d, ok)
	}
}

func TestSelectEntitiesGraphFallsBack(t *testing.T) {
	sims := map[string]float64{}
	for k := range patSims() {
		sims[k] = 0.1
	}
	cos := map[string][2]float64{}
	for _, tp := range patTopics() {
		cos[tp.Label] = [2]float64{0.9, 0.9}
	}
	raw := patRaw()
	raw.Text = "Candles form over time."
	scorer := &fakeScorer{sim: sims, cos: cos}
	rep := runSelect(t, patResearcher(t, raw, patEntities(), patTopics(), scorer), patQuery(DefaultSelectSettings))
	if rep.Graph != nil || !strings.Contains(rep.Note, "fewer than 3") {
		t.Errorf("graph %+v note %q", rep.Graph, rep.Note)
	}
	if !slices.Contains(scorer.labels, "Microeconomics") {
		t.Errorf("themes not scored: %v", scorer.labels)
	}
}

func TestSelectEntitiesNoCentreNote(t *testing.T) {
	rep := runSelect(t, newSelectFixture().researcher(), selectQuery())
	if !strings.Contains(rep.Note, "no Wikipedia article for the keyword: about uses the page's themes") || rep.Graph != nil {
		t.Errorf("graph %+v note %q", rep.Graph, rep.Note)
	}
	if b, _ := json.Marshal(rep); strings.Contains(string(b), `"graph"`) {
		t.Errorf("report has a graph key: %s", b)
	}
}

// patWithLookupHits adds two keyword lookup hits that no page entity or
// member is: "Action trading", a 2-word centre candidate of a group that
// loses to the 3-word one, linking to Technical analysis and Gap, and
// "Trading", a 1-word hit linking to those and Chart pattern.
func patWithLookupHits() *fakeWiki {
	w := patWiki()
	w.pages["Action trading"], w.pages["Trading"] = wikiPage{qid: "Q30"}, wikiPage{qid: "Q31"}
	w.items["Q30"] = wikiItem{sitelink: "Action trading"}
	w.items["Q31"] = wikiItem{sitelink: "Trading"}
	w.articleLinks["Action trading"] = []string{"Technical analysis", "Gap (chart pattern)"}
	w.articleLinks["Trading"] = []string{"Technical analysis", "Gap (chart pattern)", "Chart pattern"}
	return w
}

func TestSelectEntitiesLookupHitJoinsAboutByGraphLinks(t *testing.T) {
	set := DefaultSelectSettings
	set.MinGraphLinksUngated = 2
	rep := runSelect(t, patResearcherWiki(t, patWithLookupHits(), patRaw(), patEntities(), patTopics(), &fakeScorer{sim: patSims()}), patQuery(set))
	for name, links := range map[string]int{"Action trading": 2, "Trading": 3} {
		i := slices.IndexFunc(rep.About, func(s Selected) bool { return s.Name == name })
		if i < 0 {
			t.Errorf("%s not in about: %v", name, selNames(rep.About))
			continue
		}
		if s := rep.About[i]; s.Source != SourceKeyword || !slices.Equal(s.QualifiedBy, []string{"exact", "graph_links"}) || s.GraphLinks != links {
			t.Errorf("%s = %+v", name, s)
		}
	}
}

func TestSelectEntitiesLookupHitBelowThresholdDropped(t *testing.T) {
	rep := runSelect(t, patResearcherWiki(t, patWithLookupHits(), patRaw(), patEntities(), patTopics(), &fakeScorer{sim: patSims()}), patQuery(DefaultSelectSettings))
	for name, links := range map[string]int{"Action trading": 2, "Trading": 3} {
		d, ok := findDropped(rep, name)
		if !ok || d.Reason != ReasonOffGraph || d.Source != SourceKeyword || d.GraphLinks == nil || *d.GraphLinks != links {
			t.Errorf("dropped %s = %+v, want off_graph keyword with %d links", name, d, links)
		}
	}
}
