package costs

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// notPrices are amounts that bound or surround a price: a saving, an
// instalment or finance estimate, a bound ("under $20", "over $750"), a
// spending threshold ("orders of $35 and up") and the shipping fee under it.
var notPrices = regexp.MustCompile(`(?i)\b(?:save|payments? of|payments?:?|under|below|less than|over|above|more than|at least|minimum|spend|spending|orders? (?:of|over|above)|ships? (?:for|from|at)|shipping (?:is|costs?|from))\s*` + txt.PriceText)

// shownPrice is a price the page shows: its amount, and the text that shows
// it first.
type shownPrice struct {
	amount float64
	text   string
}

// shown are the distinct prices a page search engines index shows outside
// its header, footer and menus, in order. A saving ("save $50"), an
// instalment and a price bound ("under $20") are no item's price.
func shown(c *audit.Context) []shownPrice {
	if !c.Indexed() {
		return nil
	}
	var out []shownPrice
	seen := map[float64]bool{}
	for _, m := range txt.PriceAmount.FindAllStringSubmatch(notPrices.ReplaceAllString(c.Page.Main, " "), -1) {
		f, ok := txt.PriceOf(m)
		if !ok || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, shownPrice{f, strings.TrimSpace(m[0])})
	}
	return out
}

// unmarked are the prices the page shows that the draft does not carry and
// no price taker took before the check id.
func unmarked(c *audit.Context, id string) []string {
	prices := shown(c)
	if len(prices) == 0 {
		return nil
	}
	marked, taken := carriedBy(c).all, c.TakenBefore(id)
	var out []string
	for _, p := range prices {
		if !marked[p.amount] && !taken[p.amount] {
			out = append(out, p.text)
		}
	}
	return out
}
