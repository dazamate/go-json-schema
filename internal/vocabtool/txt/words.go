// Package txt holds the text helpers the audit checks share: whole-word
// matching, clipping, tag stripping, URL normalization and price reading.
package txt

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// NormWords lowercases s and turns every run of non-letters and non-digits
// into one space, so "Pre-wedding" and "pre wedding" match.
func NormWords(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

// Names reports whether the padded, normalized page text has the form as
// whole words, in the singular or the plural: "Bride" names "brides", and
// "Chinese Australians" names "Chinese-Australian".
func Names(paddedNormPage, form string) bool {
	f := NormWords(form)
	if f == "" {
		return false
	}
	stem := f
	if s, ok := strings.CutSuffix(f, "s"); ok && s != "" {
		stem = s
	}
	for _, v := range []string{f, stem, stem + "s", stem + "es"} {
		if strings.Contains(paddedNormPage, " "+v+" ") {
			return true
		}
	}
	return false
}

var tags = regexp.MustCompile(`<[^>]*>`)

// StripTags is the text of a value that may hold HTML.
func StripTags(s string) string { return html.UnescapeString(tags.ReplaceAllString(s, " ")) }

// Clip shortens s to n runes.
func Clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// Examples joins up to four examples, and says how many more there are.
func Examples(list []string) string {
	if len(list) > 4 {
		return strings.Join(list[:4], ", ") + fmt.Sprintf(" and %d more", len(list)-4)
	}
	return strings.Join(list, ", ")
}

// NormURL drops the fragment, lowercases the scheme and host, and drops a
// trailing slash, so links to one page compare equal.
func NormURL(u *url.URL) string {
	v := *u
	v.Fragment, v.RawFragment = "", ""
	v.Scheme, v.Host = strings.ToLower(v.Scheme), strings.ToLower(v.Host)
	if len(v.Path) > 1 {
		v.Path = strings.TrimSuffix(v.Path, "/")
		v.RawPath = ""
	}
	if v.Path == "/" {
		v.Path = ""
	}
	return v.String()
}

// NormLink is the normalized form of a link; text that is no URL stays as it is.
func NormLink(s string) string {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return s
	}
	return NormURL(u)
}
