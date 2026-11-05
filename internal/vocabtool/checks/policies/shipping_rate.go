package policies

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(ShippingRate{}) }

var _ audit.PriceTaker = ShippingRate{}

var shipCountry = regexp.MustCompile(`(?i)\b(?:shipping|delivery|postage|ship|orders?)\s+(?:within|to|across|in)\s+(?:the )?(Australia|AU|New Zealand|NZ|United States|USA|US|United Kingdom|UK|Canada)\b|\bfree (Australian|NZ|US|UK|Canadian) (?:shipping|delivery|postage)\b`)

// ShippingRate warns on a shipping rate a shop's page gives that no shipping
// node carries. It takes the rate and the free-shipping threshold, whether
// the draft carries them or not, so the prices checks do not ask for them as
// items' prices.
type ShippingRate struct{}

// ID is the check's ID.
func (ShippingRate) ID() string { return "policies/shipping-rate" }

// Run reports the first rate the page gives, with the country it ships to
// when the page names it.
func (ShippingRate) Run(c *audit.Context) []audit.Issue {
	s, ok := shippingOf(c)
	if !ok {
		return nil
	}
	v, f, ok := s.rate()
	if !ok || txt.HoldsNumber(s.ship, f) {
		return nil
	}
	country := "its country"
	if m := shipCountry.FindStringSubmatch(c.Page.Text); m != nil {
		country = txt.CountryCode(txt.FirstNonEmpty(m[1:]...))
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page gives a shipping rate of $%s that no node carries: add a ShippingService to the business's hasShippingService (fulfillmentType https://schema.org/FulfillmentTypeDelivery) with shippingConditions, a ShippingConditions with shippingDestination (a DefinedRegion with addressCountry %s) and shippingRate (a MonetaryAmount of %s); on a product page give its Offer the same as shippingDetails, an OfferShippingDetails", v, country, v)}
}

// TakesPrices takes the shipping rate and the free-shipping threshold.
func (ShippingRate) TakesPrices(c *audit.Context, _ audit.Taken) audit.Taken {
	s, ok := shippingOf(c)
	if !ok {
		return nil
	}
	t := audit.Taken{}
	if _, f, ok := s.rate(); ok {
		t[f] = true
	}
	if _, f, ok := s.freeOver(); ok {
		t[f] = true
	}
	return t
}
