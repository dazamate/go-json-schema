package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(BackOrder{}) }

var backOrder = regexp.MustCompile(`(?i)\bback-?order(?:ed|s)?\b`)

// BackOrder warns when a shop's page says a product is on back order and no
// Offer has the availability BackOrder.
type BackOrder struct{}

// ID is the check's ID.
func (BackOrder) ID() string { return "product/back-order" }

// Run reports the first mention of back orders.
func (BackOrder) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	m := backOrder.FindString(c.Page.Main)
	if m == "" || jsonld.KeyMember(c.Nodes, "availability", "BackOrder") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set the Offer's availability https://schema.org/BackOrder", m)}
}
