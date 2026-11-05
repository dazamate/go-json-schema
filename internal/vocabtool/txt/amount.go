package txt

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// The parts of a shown price: a currency written before the amount ("$",
// "A$", "€", "£", "₹", "Rs.", "CHF", "AUD"), one written after it ("50 €",
// "120 kr", "50 USD"), the amount with whatever grouping and decimal mark
// its country uses, and the space between them, a no-break space included.
const (
	leadCurrency  = `(?:(?:A|AU|US|NZ|C|CA|S|HK|R|MX)?\$|€|£|¥|₹|₱|₩|₦|₪|₺|₽|\bRs\.?|\b(?:AUD|USD|NZD|CAD|GBP|EUR|CHF|INR|SGD|HKD|ZAR|JPY|SEK|NOK|DKK|PLN))`
	trailCurrency = `(?:€|£|zł|(?:kr|AUD|USD|NZD|CAD|GBP|EUR|CHF|INR|SGD|HKD|ZAR|JPY|SEK|NOK|DKK|PLN)\b)`
	shownNumber   = `\d(?:[\d.,]*\d)?`
	currencySpace = `[\s\x{00a0}\x{202f}]?`
)

// PriceAmount matches a price with its currency symbol or code, capturing
// the currency and the amount: groups 1 and 2 when the currency comes first,
// 3 and 4 (amount, currency) when it comes after. PriceOf reads a match.
var PriceAmount = regexp.MustCompile(`(` + leadCurrency + `)` + currencySpace + `(` + shownNumber + `)|(` + shownNumber + `)` + currencySpace + `(` + trailCurrency + `)`)

// PriceText is the pattern of a price written with its currency, for
// building other patterns: "$1,150", "A$12.95", "€1.234,56", "50 €".
const PriceText = `(?:` + leadCurrency + currencySpace + shownNumber + `|` + shownNumber + currencySpace + trailCurrency + `)`

// commaCurrencies write decimals with a comma, so a point before three
// digits groups thousands: €1.234 is 1234, where $3.459 is 3.459.
var commaCurrencies = map[string]bool{"€": true, "EUR": true, "kr": true, "SEK": true, "NOK": true, "DKK": true, "zł": true, "PLN": true, "R$": true}

var (
	commaGroups = regexp.MustCompile(`^(?:\d{1,3}(?:,\d{3})+|\d{1,2}(?:,\d{2})+,\d{3})$`) // 1,234,567 or the lakh 1,00,000
	pointGroups = regexp.MustCompile(`^\d{1,3}(?:\.\d{3})+$`)
	plainDigits = regexp.MustCompile(`^\d+$`)
)

// plainAmount is a shown amount as a plain number, its decimals as written:
// "1,234.56" and "1.234,56" give "1234.56", "1,00,000" gives "100000".
func plainAmount(currency, num string) (string, bool) {
	whole, dec := num, ""
	comma, point := strings.LastIndex(num, ","), strings.LastIndex(num, ".")
	switch {
	case comma >= 0 && point >= 0:
		mark := max(comma, point)
		whole, dec = num[:mark], num[mark+1:]
	case comma >= 0 && !commaGroups.MatchString(num):
		whole, dec = num[:comma], num[comma+1:]
	case point >= 0 && !(pointGroups.MatchString(num) && (commaCurrencies[currency] || strings.Count(num, ".") > 1)):
		whole, dec = num[:point], num[point+1:]
	}
	if !commaGroups.MatchString(whole) && !pointGroups.MatchString(whole) && !plainDigits.MatchString(whole) {
		return "", false
	}
	whole = Digits(whole)
	if dec == "" {
		return whole, true
	}
	if !plainDigits.MatchString(dec) || (comma > point && len(dec) > 2) {
		return "", false
	}
	return whole + "." + dec, true
}

// matchAmount is a PriceAmount match's amount as a plain number.
func matchAmount(m []string) (string, bool) {
	if m[2] != "" {
		return plainAmount(m[1], m[2])
	}
	return plainAmount(m[4], m[3])
}

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
	s, ok := matchAmount(m)
	if !ok {
		return 0, false
	}
	return Amount(s)
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
	if s, ok := matchAmount(m); ok {
		return s
	}
	return price
}

// NumberIn matches each plain number in a text: 12, 6.5.
var NumberIn = regexp.MustCompile(`\d+(?:\.\d+)?`)
