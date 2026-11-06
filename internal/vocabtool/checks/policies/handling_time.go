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

// A handling time's span and unit: "1-2 business days", "3 weeks", "24 hours".
const (
	handleNum  = `(\d+)(?:\s*(?:-|–|to)\s*(\d+))?`
	handleUnit = `(hours?|business days?|working days?|days?|weeks?)\b`
)

// handling finds a handling time, in three groups of low, high and unit per
// way to say it: "dispatched within 1-2 business days", "ships in 3-6 days"
// (not "processed in", which is also how a refund is said), "processing time:
// 1-3 days", "allow up to 2 business days to fulfill an order", "takes 1-2
// days to process".
var handling = regexp.MustCompile(`(?i)\b(?:posted|dispatch(?:ed|es)?|despatch(?:ed|es)?|shipped|ships?|sent|processed)(?: out)?(?:\s+(?:typically|usually|normally|generally))?\s+within\s+` + handleNum + `\s*` + handleUnit +
	`|\b(?:posted|dispatch(?:ed|es)?|despatch(?:ed|es)?|shipped|ships?|sent)(?: out)?(?:\s+(?:typically|usually|normally|generally))?\s+in\s+` + handleNum + `\s*` + handleUnit +
	`|\bprocessing time(?: is| of)?:?\s*(?:up to\s+)?` + handleNum + `\s*` + handleUnit +
	`|\ballow(?:ing)?(?: up to)?\s+` + handleNum + `\s*` + handleUnit + `\s+(?:for|to)\s+(?:\w+\s+){0,3}?(?:process\w*|dispatch\w*|despatch\w*|fulfil\w*|ship\b)` +
	`|\btakes?(?: up to)?\s+` + handleNum + `\s*` + handleUnit + `\s+to\s+(?:process|dispatch|despatch|fulfil|fulfill|prepare|ship)\b`)

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
		var lo, hi, unit string
		for i := 1; i+2 < len(m); i += 3 {
			if m[i] != "" {
				lo, hi, unit = m[i], m[i+1], m[i+2]
				break
			}
		}
		maxV, _ := strconv.Atoi(txt.OrDefault(hi, lo))
		d := maxV
		switch u := strings.ToLower(unit); {
		case strings.HasPrefix(u, "hour"):
			d = (maxV + 23) / 24
		case strings.HasPrefix(u, "week"):
			d = maxV * 7
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
