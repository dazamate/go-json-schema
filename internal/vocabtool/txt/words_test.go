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
}
