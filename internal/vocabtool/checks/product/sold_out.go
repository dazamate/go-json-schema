package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(SoldOut{}) }

var (
	soldOut     = regexp.MustCompile(`(?i)\b(sold out|out of stock)\b`)
	soldOutSkip = regexp.MustCompile(`(?i)(hide|show|include|exclude|without)\s*$`)
)

// SoldOut warns when a shop's page shows a product as sold out and no Offer
// has the availability SoldOut or OutOfStock. A filter's label ("Hide sold
// out") is no product's state.
type SoldOut struct{}

// ID is the check's ID.
func (SoldOut) ID() string { return "product/sold-out" }

// Run reports the first sold-out notice that is no filter's label.
func (SoldOut) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	main := c.Page.Main
	for _, loc := range soldOut.FindAllStringIndex(main, -1) {
		if soldOutSkip.MatchString(main[:loc[0]]) {
			continue
		}
		if jsonld.KeyMember(c.Nodes, "availability", "SoldOut", "OutOfStock") {
			return nil
		}
		return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows %q: set that product's Offer availability https://schema.org/SoldOut (or OutOfStock)", main[loc[0]:loc[1]])}
	}
	return nil
}
