package txt

import "testing"

// TestNormWords checks that punctuation becomes one space.
func TestNormWords(t *testing.T) {
	if got, want := NormWords("Pre-wedding"), "pre wedding"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestNames checks that a form matches its plural as whole words.
func TestNames(t *testing.T) {
	if !Names(" brides ", "Bride") {
		t.Error(`Names(" brides ", "Bride") = false, want true`)
	}
	pad := func(s string) string { return " " + NormWords(s) + " " }
	for _, r := range []struct {
		page, form string
		want       bool
	}{
		// officeworks.com.au, tutorfinder.com.au, kasaustralia.com.au and
		// firstlookweddings.co.uk name these only in the plural, or with
		// an ampersand
		{"Beauty & Wellness Batteries Energizer Batteries All Batteries", "Battery", true},
		{"tutors at universities across Australia", "University", true},
		{"Moving Accessories String Utility Knives Water", "Knife", true},
		{"Ceramics & Vases Throws Candles", "Ceramics and vases", true},
		{"Book your photographer & videographer", "Photographer and videographer", true},
		{"Book your photographer and videographer", "Photographer & videographer", true},
		{"two buses a day", "Bus", true},
		{"a Wi-Fi battery pack", "Battery", true},
		// no other word is a plural of the form
		{"the toys and days of summer", "Toy", true},
		{"a busy day", "Bus", false},
		{"caravans for hire", "Car", false},
		{"lives on the coast", "Live music", false},
	} {
		if got := Names(pad(r.page), r.form); got != r.want {
			t.Errorf("Names(%q, %q) = %v, want %v", r.page, r.form, got, r.want)
		}
	}
}
