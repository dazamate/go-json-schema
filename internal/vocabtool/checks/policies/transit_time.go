package policies

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(TransitTime{}) }

// A time in days or weeks: "3", "2-6" or "2 to 7", then its unit.
const (
	spanNum  = `(\d+)(?:\s*(?:-|–|to)\s*(\d+))?`
	spanUnit = `(business days?|working days?|days?|weeks?)\b`
)

// transit finds a transit time: a word for delivery then its time ("delivery
// in 2-6 business days", "arrive within 3 business days", "delivery takes 3-5
// days"), or "allow 5-10 business days for delivery".
var transit = regexp.MustCompile(`(?i)\b(?:delivery|delivered|transit|arrives?|arrival)(?: time)?(?:(?:\s+(?:takes?|will take|is|in|within|of|approximately|about|around|usually|typically))+:?|\s*:)\s*(?:up to\s+)?` + spanNum + `\s*` + spanUnit +
	`|\b(?:delivery|delivered|transit|arrives?|arrival)(?: time)?:?\s*(\d+)\s*(?:-|–|to)\s*(\d+)\s*` + spanUnit +
	`|\ballow(?:ing)?(?: up to)?\s+` + spanNum + `\s*` + spanUnit + `\s+for\s+(?:\w+\s+){0,2}?(?:delivery|arrival|transit|shipment|shipping)\b`)

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
	m, lo, hi, days := transitIn(s.text)
	if m == "" {
		return nil
	}
	vals := jsonld.Under(c.Nodes, "transitTime")
	loV, _ := txt.Amount(lo)
	hiV, _ := txt.Amount(hi)
	if txt.HoldsNumber(vals, loV) && txt.HoldsNumber(vals, hiV) {
		return nil
	}
	span := lo + " to " + hi + " " + days
	if lo == hi {
		span = lo + " " + days
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q but no transitTime carries it: set transitTime on the ShippingConditions (a ServicePeriod with businessDays and a duration of %s) or in the Offer's ShippingDeliveryTime (a QuantitativeValue with minValue %s, maxValue %s and unitCode DAY)", m, span, lo, hi)}
}

// transitIn is the first transit time in the text that is a range, or else
// the first that is one number, as written, with its low and high in days.
func transitIn(text string) (said, lo, hi, unit string) {
	for _, m := range transit.FindAllStringSubmatch(text, -1) {
		var a, b, u string
		for i := 1; i+2 < len(m); i += 3 {
			if m[i] != "" {
				a, b, u = m[i], m[i+1], m[i+2]
				break
			}
		}
		if b == "" {
			b = a
		}
		if strings.HasPrefix(strings.ToLower(u), "week") {
			a, b, u = days(a), days(b), "days"
		} else {
			u = "days"
		}
		if said == "" || (lo == hi && a != b) {
			said, lo, hi, unit = m[0], a, b, u
		}
		if lo != hi {
			break
		}
	}
	return said, lo, hi, unit
}

// days is a number of weeks as a number of days.
func days(weeks string) string {
	n, _ := strconv.Atoi(weeks)
	return strconv.Itoa(n * 7)
}
