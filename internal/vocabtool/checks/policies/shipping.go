// Package policies holds the policy checks: they hold a shop's markup to the
// terms its pages state, such as its shipping rates, free-shipping threshold
// and times.
package policies

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

const money = `(?:A\$|AU\$|NZ\$|US\$|\$|£|€)\s?(\d+(?:\.\d{1,2})?)`

var (
	freeShipping  = regexp.MustCompile(`(?i)\bfree (?:\w+ ){0,2}(?:shipping|delivery|postage)(?: (?:on|for) (?:all )?orders)?(?: (?:over|above|of|from)| when you spend)\s*` + money)
	shippingRate  = regexp.MustCompile(`(?i)\b(?:shipping|delivery|postage)(?: rate| cost| fee)?(?: within \w+)?(?: is)?(?: a)?(?: flat(?: rate)?(?: of)?)?\s*:?\s*` + money + `|` + money + ` flat[- ]rate (?:shipping|delivery|postage)`)
	freeOtherwise = regexp.MustCompile(`(?i)\bfree (?:\w+ ){0,2}(?:over|above|from)\s*` + money + `,?\s*otherwise\s*` + money)
)

// shipping is what a shop's page states about its shipping.
type shipping struct {
	text string   // the page's text, with "free over $60, otherwise $7.95" spelt out as a rate and a threshold
	ship []string // the scalars under the draft's shipping nodes
}

// shippingOf is the shipping a page search engines index of a shop states;
// ok is false on any other page.
func shippingOf(c *audit.Context) (s shipping, ok bool) {
	if !c.Indexed() || !c.Page.Sells() {
		return shipping{}, false
	}
	s.text = c.Page.Text
	if m := freeOtherwise.FindStringSubmatch(s.text); m != nil {
		s.text = s.text + " Shipping rate $" + m[2] + ". Free shipping over $" + m[1] + "."
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

// rate is the shipping rate the page gives, as written and as an amount (0
// when it does not read as one).
func (s shipping) rate() (string, float64, bool) {
	m := shippingRate.FindStringSubmatch(s.text)
	if m == nil {
		return "", 0, false
	}
	v := txt.FirstNonEmpty(m[1:]...)
	f, _ := txt.Amount(v)
	return v, f, true
}

// freeOver is the order value over which the page says shipping is free, as
// written and as an amount (0 when it does not read as one).
func (s shipping) freeOver() (string, float64, bool) {
	m := freeShipping.FindStringSubmatch(s.text)
	if m == nil {
		return "", 0, false
	}
	f, _ := txt.Amount(m[1])
	return m[1], f, true
}
