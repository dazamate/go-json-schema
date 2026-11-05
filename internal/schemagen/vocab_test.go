package schemagen

import (
	"fmt"
	"slices"
	"sync"
	"testing"
)

const vocabPath = "vocab/schemaorg-all-https.jsonld"

func loadVocab(t *testing.T) *Vocab {
	t.Helper()
	v, err := Load(vocabPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return v
}

func TestLoadKeepsOnlySchemaTerms(t *testing.T) {
	v := loadVocab(t)
	if _, ok := v.Props["knowsAbout"]; !ok {
		t.Fatal("knowsAbout missing")
	}
	for name := range v.Props {
		if slices.Contains([]rune(name), ':') {
			t.Fatalf("non-schema term kept: %s", name)
		}
	}
	if len(v.Props) < 1500 {
		t.Fatalf("only %d properties loaded", len(v.Props))
	}
}

func TestAncestorsFollowMultipleParents(t *testing.T) {
	v := loadVocab(t)
	got := v.Ancestors("HowToSection")
	want := []string{"CreativeWork", "HowToSection", "Intangible", "ItemList", "ListItem", "Thing"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Ancestors caches what it works out, so concurrent lookups of classes it has
// not seen yet must not race: run it under go test -race.
func TestAncestorsConcurrentLookups(t *testing.T) {
	v := loadVocab(t)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				class := fmt.Sprintf("Unknown%d", i)
				if got := v.Ancestors(class); !slices.Equal(got, []string{class}) {
					t.Errorf("Ancestors(%q) = %v", class, got)
				}
			}
		}()
	}
	wg.Wait()
}

func TestApplicablePropertiesIncludeInherited(t *testing.T) {
	v := loadVocab(t)
	for class, want := range map[string]int{"Recipe": 148, "ProfessionalService": 129} {
		if got := len(v.Applicable(class)); got != want {
			t.Errorf("%s: %d properties, want %d", class, got, want)
		}
	}
	if !slices.Contains(v.Applicable("ProfessionalService"), "knowsAbout") {
		t.Error("ProfessionalService lacks knowsAbout (inherited from Organization)")
	}
}

func TestDatatypesBecomeValueTypes(t *testing.T) {
	v := loadVocab(t)
	for _, name := range []string{"Text", "URL", "Number", "Integer", "Float", "Boolean", "Date", "DateTime", "Time", "Duration", "Energy", "Mass", "Distance", "Quantity"} {
		if v.IsStruct(name) {
			t.Errorf("%s should be a value type, not a struct", name)
		}
	}
	for _, name := range []string{"Thing", "Recipe", "Person", "PropertyValue", "RestrictedDiet"} {
		if !v.IsStruct(name) {
			t.Errorf("%s should be a struct", name)
		}
	}
}

func TestEnumMembersKnowTheirEnumeration(t *testing.T) {
	v := loadVocab(t)
	m, ok := v.Members["VeganDiet"]
	if !ok {
		t.Fatal("VeganDiet missing")
	}
	if !slices.Contains(m.Enums, "RestrictedDiet") {
		t.Fatalf("VeganDiet enums = %v", m.Enums)
	}
}

func TestPlanPlacesEveryApplicablePropertyOnce(t *testing.T) {
	v := loadVocab(t)
	for _, class := range v.StructClasses() {
		p := v.Plan(class)
		seen := map[string]int{}
		for _, b := range p.Blocks {
			for _, prop := range v.BlockProps(b) {
				seen[prop]++
			}
		}
		for _, prop := range p.Traits {
			seen[prop]++
		}
		for _, prop := range v.Applicable(class) {
			if seen[prop] != 1 {
				t.Fatalf("%s.%s placed %d times", class, prop, seen[prop])
			}
		}
		if len(seen) != len(v.Applicable(class)) {
			t.Fatalf("%s: plan has %d properties, applicable %d", class, len(seen), len(v.Applicable(class)))
		}
	}
}

func TestPlanUsesSharedTraitForMultiDomainProperty(t *testing.T) {
	v := loadVocab(t)
	for _, class := range []string{"Person", "Organization", "ProfessionalService"} {
		if !slices.Contains(v.Plan(class).Traits, "knowsAbout") {
			t.Errorf("%s does not embed the knowsAbout trait", class)
		}
	}
	if !slices.Contains(v.Plan("Recipe").Blocks, "Thing") {
		t.Error("Recipe does not embed ThingProps")
	}
	if !slices.Contains(v.BlockProps("Thing"), "sameAs") {
		t.Error("sameAs not in ThingProps")
	}
}

func TestEmbeddedVocabMatchesFile(t *testing.T) {
	v, err := Embedded()
	if err != nil {
		t.Fatalf("Embedded: %v", err)
	}
	if got, want := len(v.Props), len(loadVocab(t).Props); got != want {
		t.Fatalf("embedded vocab has %d properties, file has %d", got, want)
	}
}

func TestGoNameMatchesGeneratedFields(t *testing.T) {
	for prop, want := range map[string]string{"sameAs": "SameAs", "contentUrl": "ContentURL", "recipeIngredient": "RecipeIngredient"} {
		if got := GoName(prop); got != want {
			t.Errorf("GoName(%q) = %q, want %q", prop, got, want)
		}
	}
}
