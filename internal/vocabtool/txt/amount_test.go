package txt

import (
	"regexp"
	"testing"
)

// TestAmount checks that Amount reads a price with a thousands separator.
func TestAmount(t *testing.T) {
	got, ok := Amount("1,250.50")
	if !ok || got != 1250.5 {
		t.Errorf("got %v, %v, want 1250.5, true", got, ok)
	}
}

// TestPriceOf checks that a shown price reads as its amount whichever
// country wrote it: the symbol before or after, a comma or a point for the
// decimals, Indian lakh grouping, and a no-break space after the symbol.
func TestPriceOf(t *testing.T) {
	for _, c := range []struct {
		text string
		want float64
	}{
		{"$1,234", 1234},
		{"$1,234.56", 1234.56},
		{"A$12.95", 12.95},
		{"from $5 to $10", 5},
		{"€1.234,56", 1234.56},
		{"€1.234", 1234},
		{"€12,50", 12.5},
		{"1,23 €", 1.23},
		{"50 €", 50},
		{"₹1,00,000", 100000},
		{"Rs. 1,299", 1299},
		{"£\u00a050", 50},
		{"CHF 120", 120},
		{"120 kr per hour", 120},
		{"USD 50", 50},
		{"50 USD", 50},
		{"R$ 99,90", 99.9},
		{"Qty 1 €80.00", 80},
		{"2 £15 each", 15},
	} {
		m := PriceAmount.FindStringSubmatch(c.text)
		if m == nil {
			t.Errorf("%q: no price found", c.text)
			continue
		}
		if got, ok := PriceOf(m); !ok || got != c.want {
			t.Errorf("%q: got %v, %v, want %v", c.text, got, ok, c.want)
		}
	}
}

// TestPriceAmountSkipsNonPrices checks that a number with no currency, or one
// followed by a word that only starts like a currency code, is no price.
func TestPriceAmountSkipsNonPrices(t *testing.T) {
	for _, s := range []string{"Top 10 tips", "50 kraft boxes", "2026 edition", "3 Euros"} {
		if m := PriceAmount.FindString(s); m != "" {
			t.Errorf("%q: found price %q", s, m)
		}
	}
}

// TestDigitsOf checks that a shown price comes back as a plain number, with
// its cents as written.
func TestDigitsOf(t *testing.T) {
	for in, want := range map[string]string{
		"$1,150":    "1150",
		"A$40.50":   "40.50",
		"€1.234,56": "1234.56",
		"₹1,00,000": "100000",
		"no price":  "no price",
	} {
		if got := DigitsOf(in); got != want {
			t.Errorf("DigitsOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestPriceTextMatchesWholePrice checks that PriceText, which other patterns
// are built from, takes in each form of price whole.
func TestPriceTextMatchesWholePrice(t *testing.T) {
	re := regexp.MustCompile(PriceText)
	for _, s := range []string{"$1,150", "A$12.95", "€1.234,56", "1,23 €", "₹1,00,000", "£\u00a050", "CHF 120", "Rs. 1,299"} {
		if got := re.FindString(s); got != s {
			t.Errorf("PriceText on %q matched %q", s, got)
		}
	}
}
