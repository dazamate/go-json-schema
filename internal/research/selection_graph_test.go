package research

import (
	"fmt"
	"reflect"
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
