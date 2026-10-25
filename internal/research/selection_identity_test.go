package research

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/azurelang"
)

// wdItem builds a Wikidata item with an enwiki sitelink and claims given as
// prop -> values; a "p:" prefix marks a preferred-rank value, Q-values are
// item IDs.
func wdItem(t *testing.T, sitelink string, claims map[string][]string) wdEntity {
	t.Helper()
	cl := map[string]any{}
	for p, vals := range claims {
		var list []any
		for _, v := range vals {
			rank := "normal"
			if strings.HasPrefix(v, "p:") {
				rank, v = "preferred", v[2:]
			}
			var dv any = v
			if strings.HasPrefix(v, "Q") {
				dv = map[string]any{"entity-type": "item", "id": v}
			}
			list = append(list, map[string]any{"mainsnak": map[string]any{"datavalue": map[string]any{"value": dv}}, "rank": rank})
		}
		cl[p] = list
	}
	ent := map[string]any{"id": "Q1", "claims": cl}
	if sitelink != "" {
		ent["sitelinks"] = map[string]any{"enwiki": map[string]any{"site": "enwiki", "title": sitelink}}
	}
	b, _ := json.Marshal(ent)
	var it wdEntity
	if err := json.Unmarshal(b, &it); err != nil {
		t.Fatal(err)
	}
	return it
}

func TestCheckIdentityPasses(t *testing.T) {
	it := wdItem(t, "Moving company", map[string][]string{"P646": {"/m/0bwxkd"}, "P2671": {"/g/1"}})
	id, reason := checkIdentity(it, true, "en", "Moving_company", "/m/0bwxkd")
	if reason != "" || !id.KGMIDChecked || id.KGMID != "/m/0bwxkd" || id.GoogleKGID != "/g/1" || id.Title != "Moving company" {
		t.Errorf("id = %+v, reason %q", id, reason)
	}
}

func TestCheckIdentityTitleMismatch(t *testing.T) {
	it := wdItem(t, "Stairwell", nil)
	if _, reason := checkIdentity(it, true, "en", "Stairs", ""); reason != ReasonIdentityMismatch {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckIdentityGoogleIDMismatch(t *testing.T) {
	it := wdItem(t, "A", map[string][]string{"P646": {"/m/1"}})
	if _, reason := checkIdentity(it, true, "en", "A", "/m/2"); reason != ReasonIdentityMismatch {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckIdentityUncheckedWhenOneSideLacksTheID(t *testing.T) {
	with := wdItem(t, "A", map[string][]string{"P646": {"/m/1"}})
	id, reason := checkIdentity(with, true, "en", "A", "")
	if reason != "" || id.KGMIDChecked || id.KGMID != "/m/1" {
		t.Errorf("no freebase: %+v %q", id, reason)
	}
	without := wdItem(t, "A", nil)
	id, reason = checkIdentity(without, true, "en", "A", "/m/2")
	if reason != "" || id.KGMIDChecked || id.KGMID != "" {
		t.Errorf("no P646: %+v %q", id, reason)
	}
}

func TestCheckIdentityNoBorrowedGoogleID(t *testing.T) {
	it := wdItem(t, "A", map[string][]string{"P694": {"Q9"}})
	id, reason := checkIdentity(it, true, "en", "A", "")
	if reason != "" || id.KGMID != "" || id.GoogleKGID != "" {
		t.Errorf("id = %+v, reason %q", id, reason)
	}
}

func TestCheckIdentityRejectsNonArticles(t *testing.T) {
	for _, class := range []string{"Q4167410", "Q4167836", "Q13406463"} {
		it := wdItem(t, "A", map[string][]string{"P31": {"Q5", class}})
		if _, reason := checkIdentity(it, true, "en", "A", ""); reason != ReasonNotAnArticle {
			t.Errorf("%s: reason = %q", class, reason)
		}
	}
}

// Some category items, such as Q7012234 (Category:Marketing), are an
// instance of Q59541917 rather than Q4167836.
func TestCheckIdentityRejectsCategoryPages(t *testing.T) {
	it := wdItem(t, "Marketing", map[string][]string{"P31": {"Q59541917"}})
	if _, reason := checkIdentity(it, true, "en", "", ""); reason != ReasonNotAnArticle {
		t.Errorf("Q59541917: reason = %q", reason)
	}
	it = wdItem(t, "Category:Marketing", nil)
	if _, reason := checkIdentity(it, true, "en", "", ""); reason != ReasonNotAnArticle {
		t.Errorf("Category: sitelink: reason = %q", reason)
	}
}

func TestCheckIdentityNoSitelink(t *testing.T) {
	if _, reason := checkIdentity(wdItem(t, "", nil), true, "en", "A", ""); reason != ReasonIdentityMismatch {
		t.Errorf("no sitelink: reason = %q", reason)
	}
	if _, reason := checkIdentity(wdEntity{}, false, "en", "A", ""); reason != ReasonIdentityMismatch {
		t.Errorf("not found: reason = %q", reason)
	}
}

func TestCheckIdentityThemeTakesTheSitelinkTitle(t *testing.T) {
	it := wdItem(t, "Office supplies", nil)
	id, reason := checkIdentity(it, true, "en", "", "")
	if reason != "" || id.Title != "Office supplies" {
		t.Errorf("id = %+v, reason %q", id, reason)
	}
}

func TestCheckIdentityClassesStopAtTwoPreferredFirst(t *testing.T) {
	it := wdItem(t, "A", map[string][]string{"P31": {"Q1", "p:Q2"}, "P279": {"Q3"}})
	id, _ := checkIdentity(it, true, "en", "A", "")
	if want := []string{"Q2", "Q1"}; !slices.Equal(id.Classes, want) {
		t.Errorf("classes = %v", id.Classes)
	}
	it = wdItem(t, "A", map[string][]string{"P31": {"Q1"}, "P279": {"Q3", "Q4"}})
	id, _ = checkIdentity(it, true, "en", "A", "")
	if want := []string{"Q1", "Q3"}; !slices.Equal(id.Classes, want) {
		t.Errorf("classes = %v", id.Classes)
	}
}

func thingJSON(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func anyStrs(v any) []string {
	var out []string
	if x, ok := v.(string); ok { // a single value marshals as a scalar
		return []string{x}
	}
	if l, ok := v.([]any); ok {
		for _, x := range l {
			out = append(out, x.(string))
		}
	}
	return out
}

func TestEntityThingSameAsIsExactlyTheFiveFamilies(t *testing.T) {
	s := Selected{Name: "Billiard table", QID: "Q1", KGMID: "/m/1", GoogleKGID: "/g/2"}
	got := anyStrs(thingJSON(t, entityThing("en", s, nil))["sameAs"])
	want := []string{
		"https://en.wikipedia.org/wiki/Billiard_table",
		"https://www.wikidata.org/wiki/Q1",
		"https://www.google.com/search?kgmid=/m/1",
		"https://www.google.com/search?kgmid=/g/2",
		"http://dbpedia.org/resource/Billiard_table",
	}
	if !slices.Equal(got, want) {
		t.Errorf("en sameAs = %v", got)
	}
	got = anyStrs(thingJSON(t, entityThing("de", Selected{Name: "Billardtisch", QID: "Q1", KGMID: "/m/1", GoogleKGID: "/g/2"}, nil))["sameAs"])
	want = []string{
		"https://de.wikipedia.org/wiki/Billardtisch",
		"https://www.wikidata.org/wiki/Q1",
		"https://www.google.com/search?kgmid=/m/1",
		"https://www.google.com/search?kgmid=/g/2",
	}
	if !slices.Equal(got, want) {
		t.Errorf("de sameAs = %v", got)
	}
}

func TestEntityThingIdentifiers(t *testing.T) {
	s := Selected{Name: "X", QID: "Q1", KGMID: "/m/1", GoogleKGID: "/g/2", BingID: "b-1"}
	m := thingJSON(t, entityThing("en", s, nil))
	if _, ok := m["@id"]; ok {
		t.Error("@id present")
	}
	var got [][2]string
	for _, x := range m["identifier"].([]any) {
		pv := x.(map[string]any)
		got = append(got, [2]string{pv["propertyID"].(string), pv["value"].(string)})
	}
	want := [][2]string{{"Wikidata", "Q1"}, {"GoogleMachineID", "kg:/m/1"}, {"GoogleMachineID", "kg:/g/2"}, {"BingEntityID", "b-1"}}
	if !slices.Equal(got, want) {
		t.Errorf("identifiers = %v", got)
	}
}

func TestEntityThingAlternateNameLeavesOutTheName(t *testing.T) {
	s := Selected{Name: "Billiard table", QID: "Q1", Matched: []string{"pool tables", "Pool table", "billiard table"}}
	m := thingJSON(t, entityThing("en", s, []string{"Q2"}))
	if got, want := anyStrs(m["alternateName"]), []string{"pool tables", "Pool table"}; !slices.Equal(got, want) {
		t.Errorf("alternateName = %v", got)
	}
	if got, want := anyStrs(m["additionalType"]), []string{"https://www.wikidata.org/wiki/Q2"}; !slices.Equal(got, want) {
		t.Errorf("additionalType = %v", got)
	}
	m = thingJSON(t, entityThing("en", Selected{Name: "X", QID: "Q1"}, nil))
	if _, ok := m["alternateName"]; ok {
		t.Error("alternateName not omitted")
	}
	if _, ok := m["additionalType"]; ok {
		t.Error("additionalType not omitted")
	}
}

func TestAttachBingExactTitleOnly(t *testing.T) {
	sel := []*Selected{{Name: "Piano", QID: "Q1"}, {Name: "Organ", QID: "Q2"}, {Name: "Harp", QID: "Q3"}}
	ents := []azurelang.Entity{
		{BingID: "b-piano", URL: "https://en.wikipedia.org/wiki/Piano"},
		{BingID: "b-organ", URL: "https://en.wikipedia.org/wiki/Organs"},
		{BingID: "b-harp", URL: "https://es.wikipedia.org/wiki/Harp"},
	}
	attachBing(sel, nil, ents, "en")
	if sel[0].BingID != "b-piano" || sel[1].BingID != "" || sel[2].BingID != "" {
		t.Errorf("bing ids = %q %q %q", sel[0].BingID, sel[1].BingID, sel[2].BingID)
	}
}

func TestAttachBingP9885Disagreement(t *testing.T) {
	ents := []azurelang.Entity{{BingID: "b-1", URL: "https://en.wikipedia.org/wiki/Piano"}}
	sel := []*Selected{{Name: "Piano", QID: "Q1"}}
	notes := attachBing(sel, map[string]string{"Q1": "other"}, ents, "en")
	if sel[0].BingID != "" || sel[0].BingChecked || len(notes) != 1 || !strings.Contains(notes[0], "Piano") {
		t.Errorf("disagree: %+v notes %v", sel[0], notes)
	}
	sel = []*Selected{{Name: "Piano", QID: "Q1"}}
	notes = attachBing(sel, map[string]string{"Q1": "b-1"}, ents, "en")
	if sel[0].BingID != "b-1" || !sel[0].BingChecked || len(notes) != 0 {
		t.Errorf("agree: %+v notes %v", sel[0], notes)
	}
}

func TestArticleItemsRanksPreferredFirst(t *testing.T) {
	raw := json.RawMessage(`{"id":"Q1","sitelinks":{"enwiki":{"site":"enwiki","title":"A"}},"claims":{"P646":[
		{"mainsnak":{"datavalue":{"value":"/m/old"}},"rank":"normal"},
		{"mainsnak":{"datavalue":{"value":"/m/dep"}},"rank":"deprecated"},
		{"mainsnak":{"datavalue":{"value":"/m/new"}},"rank":"preferred"}]}}`)
	w := &fakeWiki{rawItems: map[string]json.RawMessage{"Q1": raw}}
	r := &resolver{ctx: context.Background(), p: w.page(), lang: "en"}
	if err := r.open(); err != nil {
		t.Fatal(err)
	}
	items, err := r.articleItems([]string{"Q1", "Q1"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := items["Q1"].ranked("P646"), []string{"/m/new", "/m/old"}; !slices.Equal(got, want) {
		t.Errorf("ranked = %v", got)
	}
	if items["Q1"].Sitelinks["enwiki"].Title != "A" {
		t.Errorf("sitelinks = %+v", items["Q1"].Sitelinks)
	}
}
