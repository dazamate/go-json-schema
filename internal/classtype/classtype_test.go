package classtype

import (
	"slices"
	"testing"

	"github.com/dazamate/go-json-schema/schema"
)

// TestOf types items by their P31 classes as Wikidata gave them on
// 2026-10-30, preferred rank first: instances get their schema.org type, and
// general ideas, whose P31 is a class of classes or absent, stay a Thing.
func TestOf(t *testing.T) {
	for _, tc := range []struct {
		name    string
		p31     []string
		isClass bool // has P279
		want    string
	}{
		{"Cantonese", []string{"Q33742", "Q1288568", "Q2748296"}, true, "Language"},
		{"English", []string{"Q33742", "Q1288568", "Q34770", "Q152559", "Q399495", "Q1097949", "Q20162172"}, true, "Language"},
		{"Mandarin", []string{"Q34770"}, true, "Language"},
		{"Chinese", []string{"Q152559", "Q1288568", "Q941501"}, true, "Language"},
		{"Standard Chinese", []string{"Q399495", "Q317623", "Q34770", "Q1288568", "Q250858"}, true, "Language"},
		{"Yue Chinese", []string{"Q34770", "Q1288568", "Q2748296"}, true, "Language"},
		{"Sydney", []string{"Q515", "Q200250", "Q1549591", "Q51929311", "Q11271835", "Q208511", "Q1066984"}, false, "City"},
		{"People's Republic of China", []string{"Q3624078", "Q6256", "Q842112", "Q859563", "Q1520223", "Q465613", "Q118365", "Q849866", "Q185145", "Q10711424"}, false, "Country"},
		{"East Asia", []string{"Q82794"}, false, "Place"},
		{"Central Coast", []string{"Q82794"}, false, "Place"},
		{"Blue Mountains", []string{"Q46831", "Q82794", "Q1640361"}, false, "Landform"},
		{"master of ceremonies", []string{"Q28640", "Q12737077", "Q214339"}, true, "Occupation"},
		{"photographer", []string{"Q12737077", "Q108300140"}, true, "Occupation"},
		{"disc jockey", []string{"Q66715801", "Q12737077"}, true, "Occupation"},
		// general ideas, held on P31 alone (isClass false)
		{"wedding", []string{"Q2245405", "Q82821"}, false, ""},                              // key event, tradition
		{"bride", []string{"Q954007", "Q135929693"}, false, ""},                             // affinity, term for a woman
		{"drink", []string{"Q19861951"}, false, ""},                                         // type of food or dish
		{"dance", []string{"Q45971958", "Q47728"}, false, ""},                               // performing arts genre, hobby
		{"toast", []string{"Q187931"}, false, ""},                                           // phrase
		{"Chinese Australians", []string{"Q108788952"}, false, ""},                          // ethnic group by residency
		{"marriage", []string{"Q1938208", "Q11666901", "Q108586636", "Q178706"}, false, ""}, // legal institution, nominal kinship, form of event, institution
		{"ceremony", []string{"Q82821"}, false, ""},                                         // tradition
		{"Chinese marriage", []string{"Q74817647"}, false, ""},                              // aspect in a geographic region
		{"red envelope", nil, false, ""},
		{"tea ceremony", nil, false, ""},
		// a kind of company: instance of company and service provider, subclass of business
		{"moving company", []string{"Q783794", "Q2169973"}, true, ""},
		{"a business", []string{"Q4830453"}, false, "Organization"},
	} {
		if got := Of(tc.p31, tc.isClass); got != tc.want {
			t.Errorf("%s: Of(%v, %v) = %q, want %q", tc.name, tc.p31, tc.isClass, got, tc.want)
		}
	}
}

// TestTable checks the table: every type is a schema.org type, none is
// one an entity found in text should never take, and no class is listed twice.
func TestTable(t *testing.T) {
	seen := map[string]string{}
	for _, row := range table {
		if _, ok := schema.New(row.Type); !ok {
			t.Errorf("%s is not a schema.org type", row.Type)
		}
		if slices.Contains([]string{"Thing", "Event", "Product", "CreativeWork", "DefinedTerm", "Intangible", "Class"}, row.Type) {
			t.Errorf("%s must not be in the table: a general idea stays a Thing", row.Type)
		}
		for _, c := range row.Classes {
			if prev, dup := seen[c]; dup {
				t.Errorf("%s is listed under %s and %s", c, prev, row.Type)
			}
			seen[c] = row.Type
		}
	}
}
