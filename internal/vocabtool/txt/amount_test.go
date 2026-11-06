package txt

import "testing"

// TestAmount checks that Amount reads a price with a thousands separator.
func TestAmount(t *testing.T) {
	got, ok := Amount("1,250.50")
	if !ok || got != 1250.5 {
		t.Errorf("got %v, %v, want 1250.5, true", got, ok)
	}
}
