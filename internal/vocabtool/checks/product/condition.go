package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(Condition{}) }

var conditions = []struct {
	re     *regexp.Regexp
	member string
}{
	{regexp.MustCompile(`(?i)\b(?:pre-?owned|pre-?loved|second[- ]hand|condition:\s*used)\b`), "UsedCondition"},
	{regexp.MustCompile(`(?i)\brefurbished\b`), "RefurbishedCondition"},
	{regexp.MustCompile(`(?i)\bdamaged (?:box|packaging|item)\b`), "DamagedCondition"},
}

// Condition warns when a shop's page says a product is pre-owned,
// refurbished or damaged and no Offer has that itemCondition.
type Condition struct{}

// ID is the check's ID.
func (Condition) ID() string { return "product/condition" }

// Run reports each condition the page states that no Offer's itemCondition
// names.
func (Condition) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	var out []audit.Issue
	for _, cd := range conditions {
		if m := cd.re.FindString(c.Page.Main); m != "" && !jsonld.KeyMember(c.Nodes, "itemCondition", cd.member) {
			out = append(out, audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set the Offer's itemCondition https://schema.org/%s (an OfferItemCondition)", m, cd.member))
		}
	}
	return out
}
