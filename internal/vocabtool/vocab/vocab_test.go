package vocab

import (
	"slices"
	"testing"
)

func load(t *testing.T) *Vocab {
	t.Helper()
	v, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSuggest(t *testing.T) {
	if got := load(t).Suggest(KindType, "Prodcut"); got != "Product" {
		t.Errorf("Suggest = %q, want Product", got)
	}
}

func TestSubtypesTaking(t *testing.T) {
	got := load(t).SubtypesTaking([]string{"Organization"}, "openingHours")
	if !slices.Contains(got, "LocalBusiness") {
		t.Errorf("SubtypesTaking = %v, want LocalBusiness among them", got)
	}
}

func TestIs(t *testing.T) {
	v := load(t)
	if !v.Is([]string{"Vehicle"}, "Product") {
		t.Error("Vehicle should be a Product")
	}
	if v.Is([]string{"Place"}, "Product") {
		t.Error("Place should not be a Product")
	}
}

func TestEnumOnly(t *testing.T) {
	enum, ok := load(t).EnumOnly("availability")
	if !ok || enum != "ItemAvailability" {
		t.Errorf("EnumOnly = %q, %v, want ItemAvailability, true", enum, ok)
	}
}
