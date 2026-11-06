package product

import (
	"regexp"
	"time"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(SaleEnds{}) }

var saleEnds = regexp.MustCompile(`(?i)\b(?:sale|offer|deal|price)s?\s+(?:ends?|valid until|good until|until)\s+(?:on\s+)?(\d{1,2}\s+[A-Z][a-z]+\s+\d{4}|[A-Z][a-z]+\s+\d{1,2},?\s+\d{4}|\d{4}-\d{2}-\d{2})`)

// SaleEnds warns when a shop's page gives the date a sale ends and no node
// has priceValidUntil: the sale Offer says until when its price holds.
type SaleEnds struct{}

// ID is the check's ID.
func (SaleEnds) ID() string { return "product/sale-ends" }

// Run reports the sale's end, with the date as an ISO date when it reads as
// one.
func (SaleEnds) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	m := saleEnds.FindStringSubmatch(c.Page.Main)
	if m == nil || jsonld.HasKey(c.Nodes, "priceValidUntil") {
		return nil
	}
	when := m[1]
	for _, layout := range []string{"2 January 2006", "January 2, 2006", "January 2 2006", "2006-01-02"} {
		if t, err := time.Parse(layout, m[1]); err == nil {
			when = t.Format("2006-01-02")
			break
		}
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set priceValidUntil %s on the sale Offer; a store-wide sale can also be a SaleEvent (name, startDate, endDate %s)", m[0], when, when)}
}
