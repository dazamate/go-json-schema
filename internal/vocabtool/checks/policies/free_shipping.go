package policies

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(FreeShipping{}) }

// FreeShipping warns on a free-shipping threshold a shop's page gives that
// no shipping node carries. ShippingRate takes its amount.
type FreeShipping struct{}

// ID is the check's ID.
func (FreeShipping) ID() string { return "policies/free-shipping" }

// Run reports the first threshold the page gives.
func (FreeShipping) Run(c *audit.Context) []audit.Issue {
	s, ok := shippingOf(c)
	if !ok {
		return nil
	}
	v, f, ok := s.freeOver()
	if !ok || txt.HoldsNumber(s.ship, f) {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page offers free shipping over $%s but no ShippingConditions has an orderValue of %s: add a ShippingConditions with orderValue (a MonetaryAmount with minValue %s) and shippingRate 0 to the business's ShippingService, or a ShippingRateSettings whose freeShippingThreshold is a DeliveryChargeSpecification of %s", v, v, v, v)}
}
