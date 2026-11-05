package costs

import (
	"regexp"
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

var (
	taxWords = regexp.MustCompile(`(?i)(?:\b(plus|excl(?:uding|udes?|\.)?|ex\.?|exclusive of|incl(?:uding|udes?|\.)?|inc\.?|inclusive of)\s+|(\+)\s*)(GST|VAT|tax(?:es)?|sales tax)\b|\b(GST|VAT|tax(?:es)?)\s+(included|inclusive|incl\.?|excluded|exclusive|excl\.?|extra)\b`)
	taxOut   = regexp.MustCompile(`(?i)^(?:\+|plus|ex|excl|exclusive|excluded|extra)`)
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
	t := taxWords.FindStringSubmatch(c.Page.Text)
	if t == nil {
		return "", false
	}
	return t[0], !taxOut.MatchString(txt.FirstNonEmpty(t[1], t[2], t[5]))
}
