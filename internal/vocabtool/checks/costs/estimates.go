package costs

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Estimates{}) }

var _ audit.PriceTaker = Estimates{}

var (
	spanRe    = regexp.MustCompile(priceSpan)
	costWords = regexp.MustCompile(`(?i)\b(?:costs?|estimates?|estimated|typical(?:ly)?|average|on average|ranges?|runs?|expect|roughly)\b`)
)

// Estimates warns on prices a site that sells nothing gives as costs or
// estimates of what others charge: "a 1-bedroom move costs $380–$960". Each
// is a MonetaryAmount.
type Estimates struct{}

// ID is the check's ID.
func (Estimates) ID() string { return "costs/estimates" }

// find is the prices the page gives as costs whose amounts are fresh, given
// those taken before; each takes its amount as it is found. took is every
// amount they take.
func (Estimates) find(c *audit.Context, before audit.Taken) (found []string, took audit.Taken) {
	if !costPage(c) {
		return nil, nil
	}
	took = audit.Taken{}
	main := c.Page.Main
	for _, m := range spanRe.FindAllStringSubmatchIndex(main, -1) {
		span := main[m[2]:m[3]]
		if !fresh(c, span, before, took) || !costWords.MatchString(clause(main, m[0], 80)) {
			continue
		}
		took.Take(span)
		found = append(found, span)
	}
	return found, took
}

// Run reports the estimates in one issue.
func (e Estimates) Run(c *audit.Context) []audit.Issue {
	found, _ := e.find(c, c.TakenBefore(e.ID()))
	if len(found) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page estimates costs (%s): mark each up as a MonetaryAmount (value, or minValue and maxValue, currency) on the thing whose cost it is, such as a Service's offers priceSpecification or a PropertyValue, when it is what others charge; give the business's own price as an Offer with seller the business", txt.Examples(found))}
}

// TakesPrices takes the amounts of the estimates it reports.
func (e Estimates) TakesPrices(c *audit.Context, before audit.Taken) audit.Taken {
	_, took := e.find(c, before)
	return took
}
