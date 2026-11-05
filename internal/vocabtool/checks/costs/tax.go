package costs

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

var (
	taxWords = regexp.MustCompile(`(?i)(?:\b(plus|excl(?:uding|udes?|\.)?|ex\.?|exclusive of|incl(?:uding|udes?|\.)?|inc\.?|inclusive of)\s+(?:all\s+(?:applicable\s+)?)?|(\+)\s*)(GST|VAT|tax(?:es)?|sales tax)\b|\b(GST|VAT|tax(?:es)?)\s+(included|inclusive|incl\.?|excluded|exclusive|excl\.?|extra)\b`)
	// notPriceTax marks a sentence whose tax wording is about shipping or
	// import charges, not the prices: "Rates are exclusive of taxes, duties,
	// and customs fees".
	notPriceTax = regexp.MustCompile(`(?i)\b(?:shipping|delivery|postage|freight|courier|duty|duties|customs|import)\b`)
	taxOut      = regexp.MustCompile(`(?i)^(?:\+|plus|ex|excl|exclusive|excluded|extra)`)
)

// taxSaid is the first thing the page says of tax ("GST included", "plus
// VAT"), and whether it means the prices include it. said is "" when the
// page says nothing of tax, or shows no price the draft carries as the price
// of a thing for sale, which is what tax wording is about.
func taxSaid(c *audit.Context) (said string, included bool) {
	sold := carriedBy(c).sold
	if !slices.ContainsFunc(shown(c), func(p shownPrice) bool { return sold[p.amount] }) {
		return "", false
	}
	text := c.Page.Text
	for _, at := range taxWords.FindAllStringSubmatchIndex(text, -1) {
		after := text[at[1]:min(len(text), at[1]+120)]
		if k := strings.Index(after, ". "); k >= 0 {
			after = after[:k]
		}
		if notPriceTax.MatchString(clause(text, at[0], 120) + text[at[0]:at[1]] + after) {
			continue
		}
		var t []string
		for i := 0; i < len(at); i += 2 {
			g := ""
			if at[i] >= 0 {
				g = text[at[i]:at[i+1]]
			}
			t = append(t, g)
		}
		return t[0], !taxOut.MatchString(txt.FirstNonEmpty(t[1], t[2], t[5]))
	}
	return "", false
}
