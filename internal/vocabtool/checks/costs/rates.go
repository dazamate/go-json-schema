package costs

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Rates{}) }

var _ audit.PriceTaker = Rates{}

// rateQualifier is who or what a rate is per before it is per time:
// "$4 per mover per hour", "$10 per seat / month".
const rateQualifier = `person|seat|user|member|mover|worker|crew member|employee|head|guest|student|child|adult|driver|agent`

var (
	// rateUnits are the units a rate is known by, with their UN/CEFACT code.
	rateUnits = map[string]string{"hour": "HUR", "hr": "HUR", "minute": "MIN", "min": "MIN", "day": "DAY", "week": "WEE", "wk": "WEE", "month": "MON", "mo": "MON", "year": "ANN", "yr": "ANN", "annum": "ANN",
		"mile": "SMI", "mi": "SMI", "km": "KMT", "kilometre": "KMT", "kilometer": "KMT", "kg": "KGM", "kilo": "KGM", "pound": "LBR", "lb": "LBR", "oz": "ONZ", "ounce": "ONZ",
		"litre": "LTR", "liter": "LTR", "gallon": "GLL", "metre": "MTR", "meter": "MTR", "acre": "ACR", "hectare": "HAR",
		"foot": "FOT", "ft": "FOT", "square foot": "FTK", "square feet": "FTK", "sq ft": "FTK", "sq. ft": "FTK", "sqft": "FTK", "cubic foot": "FTQ", "cubic feet": "FTQ",
		"sqm": "MTK", "sq m": "MTK", "sq. m": "MTK", "square metre": "MTK", "square meter": "MTK", "square yard": "YDK", "m²": "MTK", "m2": "MTK", "ft²": "FTK", "km²": "KMK"}
	// rateWords are the countable units, which take no UN/CEFACT code but
	// C62 with their unitText: a night, a room, a visit, an appointment.
	rateWords = []string{"night", "room", "item", "box", "load", "trip", "visit", "session", "lesson", "class", "appointment", "appt", "treatment", "person", "head", "seat", "user", "member", "mover", "guest"}
	// rateRe reads a price, "a", "per" or "/", an optional qualifier and a
	// known unit, so prose such as "$20 a few times" or "$149 per additional
	// tooth" stays silent.
	rateRe = regexp.MustCompile(`(?i)` + priceSpan + `\s*(?:an?\b|per\b|/)\s*(?:(` + rateQualifier + `)\s*(?:an?\b|per\b|/)\s*)?(` + unitAlternatives() + `)`)
)

// unitAlternatives is the pattern of every known unit, longest first, each
// followed by a word boundary, or ending in ² where one cannot be.
func unitAlternatives() string {
	var names []string
	for u := range rateUnits {
		names = append(names, u)
	}
	names = append(names, rateWords...)
	slices.SortFunc(names, func(a, b string) int { return len(b) - len(a) })
	var plain, sq []string
	for _, n := range names {
		q := strings.ReplaceAll(regexp.QuoteMeta(n), `\ `, `\s+`)
		if strings.HasSuffix(n, "²") {
			sq = append(sq, q)
		} else {
			plain = append(plain, q)
		}
	}
	return strings.Join(sq, "|") + `|(?:` + strings.Join(plain, "|") + `)\b`
}

// rateUnit is the unit of a rateRe match, lower-cased with its dots and
// spaces as rateUnits spells them.
func rateUnit(m []string) string {
	return strings.ToLower(strings.Join(strings.Fields(m[3]), " "))
}

// Rates warns on prices given per unit on a site that sells nothing: an
// hour, a mile, a pound, per mover. Each is a UnitPriceSpecification.
type Rates struct{}

// ID is the check's ID.
func (Rates) ID() string { return "costs/rates" }

// find is the page's rates whose amounts are fresh, given those taken
// before, each a match of rateRe; each takes its amount as it is found, so a
// rate the page repeats is found once. took is every amount they take.
func (Rates) find(c *audit.Context, before audit.Taken) (found [][]string, took audit.Taken) {
	if !costPage(c) {
		return nil, nil
	}
	took = audit.Taken{}
	for _, m := range rateRe.FindAllStringSubmatch(c.Page.Main, -1) {
		if fresh(c, m[1], before, took) {
			took.Take(m[1])
			found = append(found, m)
		}
	}
	return found, took
}

// Run reports the rates in one issue, with the unit code of each unit.
func (r Rates) Run(c *audit.Context) []audit.Issue {
	found, _ := r.find(c, c.TakenBefore(r.ID()))
	if len(found) == 0 {
		return nil
	}
	var list, codes []string
	qualifier := ""
	for _, m := range found {
		list = append(list, m[0])
		code := rateUnits[rateUnit(m)]
		if code == "" {
			code = "C62 with unitText " + rateUnit(m)
		}
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
		if m[2] != "" && qualifier == "" {
			qualifier = strings.ToLower(m[2])
		}
	}
	ref := ""
	if qualifier != "" {
		ref = `, and referenceQuantity for a "per ` + qualifier + `" rate`
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page prices %s: mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode %s%s) on the Service or Offer it is the rate of", txt.Examples(list), strings.Join(codes, " or "), ref)}
}

// TakesPrices takes the amounts of the rates it reports.
func (r Rates) TakesPrices(c *audit.Context, before audit.Taken) audit.Taken {
	_, took := r.find(c, before)
	return took
}
