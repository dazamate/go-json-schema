// Package policies holds the policy checks: they hold a shop's markup to the
// terms its pages state. They cover its shipping (rates, free-shipping
// threshold, handling and transit times), its return policy, how it takes
// payment, where it is and when it is open, how to contact it, and the
// questions it answers. A policy's words ("a damaged item", "pay by bank
// transfer") are about others on a site that sells nothing, so the shipping,
// return and payment checks run on a shop's pages only.
package policies

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// currency is a currency symbol or code a price can start with.
const currency = `(?:A\$|AU\$|NZ\$|US\$|C\$|CA\$|AUD|NZD|USD|CAD|GBP|EUR|\$|£|€)`

// money is a price in a currency, as a group 1 amount. It starts with the
// currency (a trailing "50 EUR" is not read).
const money = currency + `\s?(\d+(?:,\d{3})*(?:\.\d{1,2})?)`

// The place free shipping is offered, and the orders it needs: "free
// shipping to Ireland on orders over", "free delivery over", "free shipping
// when you spend".
const (
	freeKind  = `(?:shipping|delivery|postage)`
	freePlace = `(?:\s+(?:to|within|in)\s+[A-Za-z][A-Za-z &.'-]{1,40}?)?`
	freeOrder = `(?:\s+(?:on|for)\s+(?:all\s+|any\s+|your\s+)?(?:orders?|purchases?)(?:\s+(?:over|above|of|from|totall?ing|worth|exceeding|more than))?|\s+(?:over|above|of|from|when you spend(?:\s+(?:over|more than))?))\s*`
)

var (
	freeShipping = regexp.MustCompile(`(?i)\bfree (?:[\w.]+ ){0,2}` + freeKind + freePlace + freeOrder + money + `|\b` + freeKind + ` (?:is )?free(?: to [A-Za-z][A-Za-z &.'-]{1,40}?)? (?:on|for|with) (?:all )?(?:orders?|purchases?)(?: (?:over|above|of|from))?\s*` + money)
	// a rate is the price after "shipping", "delivery charge", "shipping &
	// handling", "shipping from", or "ship"
	shippingRate  = regexp.MustCompile(`(?i)\b(?:shipping|delivery|postage)(?: (?:rates?|costs?|fees?|charges?))?(?: (?:&|and) handling)?(?: within \w+)?(?: (?:is|are))?(?: a)?(?: flat(?: rate)?(?: of)?)?(?: (?:from|starts? at|starting at))?\s*:?\s*` + money + `|` + money + ` flat[- ]rate (?:shipping|delivery|postage)|\bships? (?:for |at )?` + money)
	freeOtherwise = regexp.MustCompile(`(?i)\bfree (?:\w+ ){0,2}(?:over|above|from)\s*` + money + `,?\s*otherwise\s*` + money)
	// symbol is the currency a matched price is in
	symbol = regexp.MustCompile(currency)
)

// symbolOf is how the matched text writes its currency: "$" for any dollar,
// "£", "€", or the code followed by a space ("CAD ").
func symbolOf(match string) string {
	switch m := strings.ToUpper(symbol.FindString(match)); m {
	case "", "A$", "AU$", "NZ$", "US$", "C$", "CA$", "$":
		return "$"
	case "GBP", "£":
		return "£"
	case "EUR", "€":
		return "€"
	default:
		return m + " "
	}
}

// shipping is what a shop's page states about its shipping.
type shipping struct {
	text string   // the page's text, with "free over $60, otherwise $7.95" spelt out as a rate and a threshold
	ship []string // the scalars under the draft's shipping nodes
}

// shippingKey is the memo key of the page's shipping.
type shippingKey struct{}

// shippingFound is the memoized shippingOf: the shipping, and whether the
// page states any.
type shippingFound struct {
	s  shipping
	ok bool
}

// shippingOf is the shipping a page search engines index of a shop states,
// worked out once for the checks that share it; ok is false on any other
// page.
func shippingOf(c *audit.Context) (s shipping, ok bool) {
	f := audit.Memo(c, shippingKey{}, func() shippingFound {
		s, ok := findShipping(c)
		return shippingFound{s, ok}
	})
	return f.s, f.ok
}

// findShipping reads the shipping off the page and the draft.
func findShipping(c *audit.Context) (s shipping, ok bool) {
	if !c.Indexed() || !c.Page.Sells() {
		return shipping{}, false
	}
	s.text = c.Page.Text
	if m := freeOtherwise.FindStringSubmatch(s.text); m != nil {
		s.text = s.text + " Shipping rate " + symbolOf(m[0]) + m[2] + ". Free shipping over " + symbolOf(m[0]) + m[1] + "."
	}
	for _, l := range c.Nodes {
		for _, ty := range []string{"ShippingConditions", "OfferShippingDetails", "ShippingRateSettings", "ShippingService", "DeliveryChargeSpecification"} {
			if c.Is(l.N, ty) {
				s.ship = append(s.ship, jsonld.DeepScalars(l.N)...)
				break
			}
		}
	}
	return s, true
}

// rate is the shipping rate the page gives, as written, its currency's symbol and
// as an amount (0 when it does not read as one).
func (s shipping) rate() (v, sym string, f float64, ok bool) {
	m := shippingRate.FindStringSubmatch(s.text)
	if m == nil {
		return "", "", 0, false
	}
	v = txt.FirstNonEmpty(m[1:]...)
	f, _ = txt.Amount(v)
	return v, symbolOf(m[0]), f, true
}

// freeOver is the order value over which the page says shipping is free, as
// written with its currency and as an amount (0 when it does not read as
// one).
func (s shipping) freeOver() (v, sym string, f float64, ok bool) {
	m := freeShipping.FindStringSubmatch(s.text)
	if m == nil {
		return "", "", 0, false
	}
	v = txt.FirstNonEmpty(m[1:]...)
	f, _ = txt.Amount(v)
	return v, symbolOf(m[0]), f, true
}
