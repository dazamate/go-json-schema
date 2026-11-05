package txt

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// PriceAmount matches a price with its currency symbol or code, capturing
// the whole part and the cents.
var PriceAmount = regexp.MustCompile(`(?:A\$|AU\$|AUD ?|US\$|USD ?|NZ\$|NZD ?|C\$|CA\$|S\$|HK\$|€|£|¥|₹|\$)\s?(\d{1,3}(?:,\d{3})+|\d+)(?:\.(\d{1,2}))?`)

// PriceText is the pattern of a price written with its currency's symbol,
// for building other patterns: "$1,150", "A$12.95", "€40".
const PriceText = `(?:A\$|AU\$|US\$|NZ\$|C\$|CA\$|€|£|\$)\s?\d(?:[\d,]*\d)?(?:\.\d{1,2})?`

// Amount reads a price's number.
func Amount(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", ""), 64)
	return f, err == nil
}

// HoldsNumber reports a number among the scalars.
func HoldsNumber(vals []string, f float64) bool {
	return slices.ContainsFunc(vals, func(s string) bool { g, ok := Amount(s); return ok && g == f })
}

// PriceOf is the amount a PriceAmount match gives.
func PriceOf(m []string) (float64, bool) {
	if m[2] == "" {
		return Amount(m[1])
	}
	return Amount(m[1] + "." + m[2])
}

// Digits are the digits in s.
func Digits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// DigitsOf is a shown price as a plain number.
func DigitsOf(price string) string {
	m := PriceAmount.FindStringSubmatch(price)
	if m == nil {
		return price
	}
	return strings.ReplaceAll(m[1], ",", "") + map[bool]string{true: "." + m[2], false: ""}[m[2] != ""]
}
