package schemagen

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func generateInto(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	if err := Generate(loadVocab(t), dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	out := map[string][]byte{}
	files, _ := filepath.Glob(filepath.Join(dir, "*_gen.go"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(f)] = b
	}
	return out
}

func TestGenerateWritesComposedTypes(t *testing.T) {
	files := generateInto(t, t.TempDir())
	all := bytes.Join([][]byte{files["types_gen.go"], files["props_gen.go"]}, nil)
	for _, want := range []string{
		"type Recipe struct {",
		"\tThingProps\n",
		"type ThingProps struct {",
		"\tSameAs Values `json:\"sameAs,omitempty\"`",
		"type KnowsAboutTrait struct {",
		"\tKnowsAbout Values `json:\"knowsAbout,omitempty\"`",
		"func (v Recipe) MarshalJSON() ([]byte, error) {",
		"func (v Person) Ref() Ref {",
	} {
		if !bytes.Contains(all, []byte(want)) {
			t.Errorf("generated code lacks %q", want)
		}
	}
	if !bytes.Contains(files["enums_gen.go"], []byte("VeganDiet RestrictedDietMember = \"VeganDiet\"")) {
		t.Error("enum constant VeganDiet missing")
	}
	if !bytes.Contains(files["registry_gen.go"], []byte("\"Recipe\":")) {
		t.Error("registry lacks Recipe")
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	a := generateInto(t, t.TempDir())
	b := generateInto(t, t.TempDir())
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Fatalf("%s differs between runs", name)
		}
	}
}

func TestGenerateRejectsCollisionWithHandwrittenCode(t *testing.T) {
	dir := t.TempDir()
	src := "package schema\n\ntype Recipe struct{}\n"
	if err := os.WriteFile(filepath.Join(dir, "mine.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Generate(loadVocab(t), dir)
	if err == nil || !strings.Contains(err.Error(), "Recipe") {
		t.Fatalf("want collision error naming Recipe, got %v", err)
	}
}

func TestGoNames(t *testing.T) {
	for in, want := range map[string]string{
		"sameAs":     "SameAs",
		"url":        "URL",
		"contentUrl": "ContentURL",
		"propertyID": "PropertyID",
		"identifier": "Identifier",
		"3DModel":    "X3DModel",
		"gtin13":     "Gtin13",
	} {
		if got := goName(in); got != want {
			t.Errorf("goName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDocCleansMarkup(t *testing.T) {
	in := "The [[Person]] or [schema](https://schema.org) <a href=\"x\">who</a> wrote\\n it &amp; more."
	got := cleanComment(in)
	want := "The Person or schema who wrote it & more."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGenerateTypesEnumOnlyProperties(t *testing.T) {
	files := generateInto(t, t.TempDir())
	all := bytes.Join([][]byte{files["props_gen.go"], files["types_gen.go"], files["enums_gen.go"]}, nil)
	// gofmt pads runs of one-line methods into columns; compare without it.
	all = regexp.MustCompile(` {2,}`).ReplaceAll(all, []byte(" "))
	for _, want := range []string{
		// enum-only property gets a typed list
		"\tAvailability ItemAvailabilityList `json:\"availability,omitempty\"`",
		"type ItemAvailabilityEnum interface {",
		"type ItemAvailabilityList []ItemAvailabilityEnum",
		"\tInStock ItemAvailabilityMember = \"InStock\"",
		"func (ItemAvailabilityMember) isItemAvailabilityEnum() {}",
		// node form and Ref are accepted too
		"func (ItemAvailability) isItemAvailabilityEnum() {}",
		"func (Ref) isItemAvailabilityEnum() {}",
		// members of a sub-enumeration fit the parent's list
		"func (MedicalSpecialtyMember) isSpecialtyEnum() {}",
		// a member of two enumerations fits both lists
		"\tRadiography MedicalImagingTechniqueAndMedicalSpecialtyMember = \"Radiography\"",
		"func (MedicalImagingTechniqueAndMedicalSpecialtyMember) isMedicalSpecialtyEnum() {}",
		"func (MedicalImagingTechniqueAndMedicalSpecialtyMember) isMedicalImagingTechniqueEnum() {}",
		// a property that also takes text stays open
		"\tSuitableForDiet Values `json:\"suitableForDiet,omitempty\"`",
	} {
		if !bytes.Contains(all, []byte(want)) {
			t.Errorf("generated code lacks %q", want)
		}
	}
}

func TestEnumOnlyPropertiesHaveOneEnumRange(t *testing.T) {
	v := loadVocab(t)
	got := v.EnumOnlyProps()
	if len(got) != 75 {
		t.Fatalf("%d enum-only properties, want 75", len(got))
	}
	if got["availability"] != "ItemAvailability" || got["specialty"] != "Specialty" {
		t.Fatalf("availability=%q specialty=%q", got["availability"], got["specialty"])
	}
	if _, ok := got["suitableForDiet"]; ok {
		t.Fatal("suitableForDiet also takes Diet nodes; it must stay open")
	}
}
