package policies

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(HandlingTime{}) }

var handling = regexp.MustCompile(`(?i)\b(?:posted|dispatched|shipped|ships|sent|processed)(?: out)? within (\d+)(?:\s*(?:-|–|to)\s*(\d+))? (hours?|business days?|working days?|days?)\b|\bprocessing time:?\s*(\d+)\s*(?:-|–|to)\s*(\d+) (business days?|working days?|days?)\b`)

// HandlingTime warns on the handling times a shop's page states ("dispatched
// within 1-2 business days") that no handlingTime carries.
type HandlingTime struct{}

// ID is the check's ID.
func (HandlingTime) ID() string { return "policies/handling-time" }

// Run reports every handling time no handlingTime holds, in days or as the
// page gives it, with the longest in days.
func (HandlingTime) Run(c *audit.Context) []audit.Issue {
	s, ok := shippingOf(c)
	if !ok {
		return nil
	}
	vals := jsonld.Under(c.Nodes, "handlingTime")
	var said []string
	days := 0
	for _, m := range handling.FindAllStringSubmatch(s.text, -1) {
		lo, hi, unit := m[1], m[2], m[3]
		if m[4] != "" {
			lo, hi, unit = m[4], m[5], m[6]
		}
		maxV, _ := strconv.Atoi(txt.OrDefault(hi, lo))
		d := maxV
		if strings.HasPrefix(strings.ToLower(unit), "hour") {
			d = (maxV + 23) / 24
		}
		if !txt.HoldsNumber(vals, float64(maxV)) && !txt.HoldsNumber(vals, float64(d)) && !slices.Contains(said, m[0]) {
			said, days = append(said, m[0]), max(days, d)
		}
	}
	if len(said) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %s but no handlingTime carries it: set handlingTime on the ShippingService (a ServicePeriod, or a QuantitativeValue with maxValue %d and unitCode DAY) or, on a product page, in its OfferShippingDetails's deliveryTime (a ShippingDeliveryTime with handlingTime a QuantitativeValue)", txt.QuoteAll(said), days)}
}
