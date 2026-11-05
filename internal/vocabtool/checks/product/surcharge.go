package product

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Surcharge{}) }

var surcharge = regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*%\s*(?:card\s+|credit card\s+)?surcharge\b|\b(?:incur|attract)s?\s+an?\s+\d+(?:\.\d+)?\s*%\s*surcharge\b|\bsurcharge of \d+(?:\.\d+)?\s*%`)

// daySurcharge is the words of a surcharge for the day, not for the way the
// shopper pays: a weekend's or a public holiday's.
var daySurcharge = regexp.MustCompile(`(?i)week-?ends?|holidays?|sundays?|saturdays?`)

// Surcharge warns on a card surcharge a shop's page states that no
// PaymentChargeSpecification carries.
type Surcharge struct{}

// ID is the check's ID.
func (Surcharge) ID() string { return "product/surcharge" }

// Run reports the first surcharge the page states.
func (Surcharge) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	if c.Has("PaymentChargeSpecification") {
		return nil
	}
	m := ""
	for _, at := range surcharge.FindAllStringIndex(c.Page.Main, -1) {
		before := c.Page.Main[max(0, at[0]-30):at[0]]
		if i := strings.LastIndexAny(before, ".|"); i >= 0 {
			before = before[i:]
		}
		if !daySurcharge.MatchString(before + c.Page.Main[at[0]:min(len(c.Page.Main), at[1]+40)]) {
			m = c.Page.Main[at[0]:at[1]]
			break
		}
	}
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: add a PaymentChargeSpecification (appliesToPaymentMethod the cards it applies to, and the charge) to the Offer's priceSpecification", m)}
}
