// Package shop holds the checks of what a shop's pages show of the shop: the
// product cards a page lists, a listing's product count, starting price and
// sort order, the page's WebPage and the type its kind of page takes, the
// sitewide WebSite and its links to the page, the business that runs the shop
// with its type, logo, social profiles and categories.
package shop

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// listed are the product cards a page of a shop that search engines index
// lists outside its blocks of related products; nil when it lists fewer than
// two, which is no listing, or when the page is of another kind than a
// listing or a home page: a pricing page's plans, a price list's services and
// an article's example homes are cards, not a shop's products.
func listed(c *audit.Context) []*webpage.Card {
	if !c.Indexed() || !c.Page.Sells() || (c.Page.Kind != webpage.KindListing && c.Page.Kind != webpage.KindHome) {
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

// fromPrice is a starting price in any currency, as its country writes it:
// "from £1,137.00", "at ₹1,49,999", "from 1.299 €".
var fromPrice = regexp.MustCompile(`(?i)\b(?:prices? (?:start|starting) (?:at|from)|starting (?:at|from)|from only)\s+(` + txt.PriceText + `)`)

// startingPrice is what a listing says its prices start from ("Prices start
// from $99.00") and that price, or nil.
func startingPrice(c *audit.Context) []string {
	if listed(c) == nil || c.Page.Kind != webpage.KindListing {
		return nil
	}
	return fromPrice.FindStringSubmatch(c.Page.Main)
}
