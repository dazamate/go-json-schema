package policies

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(TransitTime{}) }

var transit = regexp.MustCompile(`(?i)\b(?:delivery|transit|arrives?|arrival)(?: time)?(?: in| within)?:?\s*(\d+)\s*(?:-|–|to)\s*(\d+) (business days?|working days?|days?)\b`)

// TransitTime warns on a transit time a shop's page states ("delivery in
// 2-6 business days") that no transitTime carries in full.
type TransitTime struct{}

// ID is the check's ID.
func (TransitTime) ID() string { return "policies/transit-time" }

// Run reports the first transit time when no transitTime holds both its
// ends.
func (TransitTime) Run(c *audit.Context) []audit.Issue {
	s, ok := shippingOf(c)
	if !ok {
		return nil
	}
	m := transit.FindStringSubmatch(s.text)
	if m == nil {
		return nil
	}
	vals := jsonld.Under(c.Nodes, "transitTime")
	lo, _ := txt.Amount(m[1])
	hi, _ := txt.Amount(m[2])
	if txt.HoldsNumber(vals, lo) && txt.HoldsNumber(vals, hi) {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q but no transitTime carries it: set transitTime on the ShippingConditions (a ServicePeriod with businessDays and a duration of %s to %s days) or in the Offer's ShippingDeliveryTime (a QuantitativeValue with minValue %s, maxValue %s and unitCode DAY)", m[0], m[1], m[2], m[1], m[2])}
}
