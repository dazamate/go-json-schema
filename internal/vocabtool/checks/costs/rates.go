package costs

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Rates{}) }

var (
	rateRe    = regexp.MustCompile(`(?i)` + priceSpan + `\s*(?:an?|per|/)\s*(?:(mover|person|worker|crew member)\s+(?:an?|per|/)\s*)?(hour|hr|mile|mi|pound|lb|month|day|week|square foot|sq\.? ?ft|cubic foot|foot|ft|box|room|item|load|trip|mover|person)\b`)
	rateUnits = map[string]string{"hour": "HUR", "hr": "HUR", "mile": "SMI", "mi": "SMI", "pound": "LBR", "lb": "LBR", "month": "MON", "day": "DAY", "week": "WEE",
		"foot": "FOT", "ft": "FOT", "square foot": "FTK", "sq ft": "FTK", "sq. ft": "FTK", "sqft": "FTK", "cubic foot": "FTQ"}
)

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
	perPerson := false
	for _, m := range found {
		list = append(list, m[0])
		code := rateUnits[strings.ToLower(m[3])]
		if code == "" {
			code = "C62 with unitText " + strings.ToLower(m[3])
		}
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
		perPerson = perPerson || m[2] != ""
	}
	ref := ""
	if perPerson {
		ref = `, and referenceQuantity for a "per mover" rate`
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page prices %s: mark each up as a UnitPriceSpecification (price, or minPrice and maxPrice, priceCurrency, unitCode %s%s) on the Service or Offer it is the rate of", txt.Examples(list), strings.Join(codes, " or "), ref)}
}

// TakesPrices takes the amounts of the rates it reports.
func (r Rates) TakesPrices(c *audit.Context, before audit.Taken) audit.Taken {
	_, took := r.find(c, before)
	return took
}
