package research

import (
	"slices"
	"testing"
)

func TestSingular(t *testing.T) {
	for in, want := range map[string]string{"companies": "company", "homes": "home", "boxes": "box", "rates": "rate",
		"glass": "glass", "texas": "texa", "bus": "bus", "churches": "church", "home": "home"} {
		if got := singular(in); got != want {
			t.Errorf("singular(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKeywordTitle(t *testing.T) {
	grams := keywordGrams("tulsa moving companies rates")
	for title, want := range map[string]bool{
		"Tulsa, Oklahoma":        true,  // the part before the comma
		"Moving company":         true,  // plurals aside
		"Rate (company)":         true,  // without its disambiguation
		"Tulsa County, Oklahoma": false, // a word the keyword lacks
		"Moving":                 true,
		"Oklahoma":               false,
	} {
		if got := keywordTitle(title, grams); got != want {
			t.Errorf("keywordTitle(%q) = %v, want %v", title, got, want)
		}
	}
	// Abbreviations are left to require: no title spells "la" or "dc".
	if keywordTitle("La", keywordGrams("how much does it cost to move to la")) {
		t.Error(`"La" matched the gram "la"`)
	}
	if !keywordTitle("Mobile home", keywordGrams("mobile homes moving cost")) {
		t.Error("Mobile home did not match mobile homes")
	}
}

// tulsaFixture is the select fixture on a Tulsa page: Tulsa, Oklahoma scores
// below the gate (relevance 0.1, no topic) though TextRazor is sure of it.
func tulsaFixture() *selectFixture {
	f := newSelectFixture()
	f.raw.Text = "Our moving company calculator prices a truck, stairs and a pool table in Tulsa. Email us."
	tulsa := pageMention("Tulsa, Oklahoma", "Q44989", "/m/tul", "Tulsa", 0.1)
	tulsa.Confidence = 5
	f.ents = append(f.ents, tulsa)
	f.wiki.items["Q44989"] = wikiItem{sitelink: "Tulsa, Oklahoma", claims: map[string][]string{"P646": {"/m/tul"}}}
	return f
}

func TestSelectEntitiesKeywordTitleSkipsTheGate(t *testing.T) {
	f := tulsaFixture()
	q := selectQuery()
	q.Keyword = "tulsa moving cost calculator"
	rep := runSelect(t, f.researcher(), q)
	if len(rep.About) == 0 || rep.About[0].Name != "Tulsa, Oklahoma" {
		t.Fatalf("about = %v", selNames(rep.About))
	}
	a := rep.About[0]
	if !slices.Equal(a.QualifiedBy, []string{"keyword_title"}) || !slices.Equal(a.Matched, []string{"Tulsa"}) {
		t.Errorf("Tulsa = %+v", a)
	}
	if m := thingJSON(t, a.Thing); !slices.Equal(anyStrs(m["alternateName"]), []string{"Tulsa"}) {
		t.Errorf("thing = %s", a.Thing)
	}
	if _, ok := findDropped(rep, "Tulsa, Oklahoma"); ok {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
}

func TestSelectEntitiesKeywordTitleNeedsConfidence(t *testing.T) {
	f := tulsaFixture()
	f.ents[len(f.ents)-1].Confidence = 1
	q := selectQuery()
	q.Keyword = "tulsa moving cost calculator"
	rep := runSelect(t, f.researcher(), q)
	if d, ok := findDropped(rep, "Tulsa, Oklahoma"); !ok || d.Reason != ReasonBelowRelevance {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
}

func TestSelectEntitiesRequireTakesBackADroppedEntity(t *testing.T) {
	f := newSelectFixture()
	q := selectQuery()
	q.Require = []string{"Q5", "Q5"}
	rep := runSelect(t, f.researcher(), q)
	if got := selNames(rep.About); !slices.Equal(got, []string{"Stairs", "Moving company"}) {
		t.Fatalf("about = %v", got)
	}
	if s := rep.About[0]; s.Source != SourcePage || !slices.Equal(s.QualifiedBy, []string{"required"}) {
		t.Errorf("Stairs = %+v", s)
	}
	if _, ok := findDropped(rep, "Stairs"); ok {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
}

func TestSelectEntitiesRequireMovesAMention(t *testing.T) {
	f := newSelectFixture()
	q := selectQuery()
	q.Require = []string{"Q2"}
	rep := runSelect(t, f.researcher(), q)
	if got := selNames(rep.About); !slices.Equal(got, []string{"Truck", "Moving company"}) {
		t.Fatalf("about = %v", got)
	}
	if got := selNames(rep.Mentions); !slices.Equal(got, []string{"Billiard table"}) {
		t.Errorf("mentions = %v", got)
	}
}

func TestSelectEntitiesRequireNamesFromTheItem(t *testing.T) {
	f := newSelectFixture()
	f.wiki.items["Q160151"] = wikiItem{label: "price", aliases: []string{"cost price"}, sitelink: "Price",
		claims: map[string][]string{"P646": {"/m/01d_1l"}}}
	f.wiki.items["Q61"] = wikiItem{label: "Washington, D.C.", aliases: []string{"DC"}, sitelink: "Washington, D.C."}
	q := selectQuery()
	q.Require = []string{"Q61", "Q160151"}
	rep := runSelect(t, f.researcher(), q)
	if got := selNames(rep.About); !slices.Equal(got, []string{"Price", "Moving company"}) {
		t.Fatalf("about = %v", got)
	}
	p := rep.About[0]
	if p.Source != SourceRequire || !slices.Equal(p.Matched, []string{"prices"}) || p.KGMID != "/m/01d_1l" {
		t.Errorf("Price = %+v", p)
	}
	if m := thingJSON(t, p.Thing); !slices.Equal(anyStrs(m["alternateName"]), []string{"prices"}) {
		t.Errorf("thing = %s", p.Thing)
	}
	if d, ok := findDropped(rep, "Washington, D.C."); !ok || d.Reason != ReasonNotInPageText || d.Source != SourceRequire {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
}

func TestItemNames(t *testing.T) {
	it := wdEntity{Labels: map[string]struct{ Value string }{"en": {"Washington, D.C."}},
		Aliases: map[string][]struct{ Value string }{"en": {{"DC"}, {"Washington"}, {"D.C."}}}}
	got := itemNames(it, "Washington, D.C.", "en")
	if want := []string{"Washington, D.C.", "Washington", "DC", "D.C."}; !slices.Equal(got, want) {
		t.Errorf("itemNames = %v, want %v", got, want)
	}
	if got := itemNames(wdEntity{}, "Relocation (personal)", "en"); !slices.Equal(got, []string{"Relocation (personal)", "Relocation"}) {
		t.Errorf("itemNames = %v", got)
	}
}

func TestSelectEntitiesGraphKeywordTitleNeedsNoLinks(t *testing.T) {
	w := patWiki()
	w.items["Q99"] = wikiItem{sitelink: "Price"}
	ents := append(patEntities(), pageMention("Price", "Q99", "", "Price", 0.1))
	rep := runSelect(t, patResearcherWiki(t, w, patRaw(), ents, patTopics(), &fakeScorer{sim: patSims()}), patQuery(DefaultSelectSettings))
	if got := selNames(rep.About); len(got) < 2 || got[0] != "Price action trading" || got[1] != "Price" {
		t.Fatalf("about = %v", got)
	}
	if p := rep.About[1]; !slices.Equal(p.QualifiedBy, []string{"keyword_title"}) || p.GraphLinks != 0 {
		t.Errorf("Price = %+v", p)
	}
}
