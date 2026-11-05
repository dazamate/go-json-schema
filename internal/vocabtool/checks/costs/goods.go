package costs

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Goods{}) }

var (
	goodsRe   = regexp.MustCompile(`(?i)\b(boxes|box|tape|packing paper|paper|bubble wrap|mattress bags?|moving blankets?|furniture pads?|dollies|dolly|stretch wrap|packing peanuts)\b[^$.]{0,45}?` + priceSpan + `([^.$]{0,25})`)
	goodsSign = regexp.MustCompile(`(?i)\b(?:each|apiece|a pack|per pack|a roll|per roll|published prices|retail prices|typical (?:store|retail) prices)\b`)
)

// Goods warns on goods a site that sells nothing prices across sellers:
// boxes, tape, packing paper at typical prices. Each is a Product with an
// AggregateOffer.
type Goods struct{}

// ID is the check's ID.
func (Goods) ID() string { return "costs/goods" }

// find is the goods the page prices whose amounts are fresh, given those
// taken before, each as "box $1.82–$2.69"; each takes its amount as it is
// found. took is every amount they take.
func (Goods) find(c *audit.Context, before audit.Taken) (found []string, took audit.Taken) {
	if !costPage(c) {
		return nil, nil
	}
	took = audit.Taken{}
	main := c.Page.Main
	for _, m := range goodsRe.FindAllStringSubmatchIndex(main, -1) {
		span, after := main[m[4]:m[5]], main[m[6]:m[7]]
		sentence := clause(main, m[0], 200) + main[m[0]:m[1]]
		if !fresh(c, span, before, took) || !(goodsSign.MatchString(after) || goodsSign.MatchString(sentence)) {
			continue
		}
		took.Take(span)
		found = append(found, main[m[2]:m[3]]+" "+span)
	}
	return found, took
}

// Run reports the priced goods in one issue.
func (g Goods) Run(c *audit.Context) []audit.Issue {
	found, _ := g.find(c, c.TakenBefore(g.ID()))
	if len(found) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page prices goods (%s): add a Product for each with an AggregateOffer (lowPrice, highPrice, priceCurrency, and offerCount: how many sellers or sources the range spans)", txt.Examples(found))}
}

// TakesPrices takes the amounts of the goods it reports.
func (g Goods) TakesPrices(c *audit.Context, before audit.Taken) audit.Taken {
	_, took := g.find(c, before)
	return took
}
