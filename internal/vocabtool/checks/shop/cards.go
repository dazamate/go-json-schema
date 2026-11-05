// Package shop holds the checks of what a shop's pages show: the product
// cards a page lists, a listing's product count, starting price and sort
// order, the page's WebPage and the type its kind of page takes, and the
// business that runs the shop and its type.
package shop

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// listed are the product cards a page search engines index lists outside
// its blocks of related products; nil when it lists fewer than two, which is
// no listing.
func listed(c *audit.Context) []*webpage.Card {
	if !c.Indexed() {
		return nil
	}
	var out []*webpage.Card
	for _, cd := range c.Page.ProductCards() {
		if !cd.Related {
			out = append(out, cd)
		}
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

var fromPrice = regexp.MustCompile(`(?i)\b(?:prices? (?:start|starting) (?:at|from)|starting (?:at|from)|from only)\s+((?:A\$|AU\$|\$|£|€)\s?\d+(?:\.\d{1,2})?)`)

// startingPrice is what a listing says its prices start from ("Prices start
// from $99.00") and that price, or nil.
func startingPrice(c *audit.Context) []string {
	if listed(c) == nil || c.Page.Kind != webpage.KindListing {
		return nil
	}
	return fromPrice.FindStringSubmatch(c.Page.Main)
}
