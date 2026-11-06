package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(PreOrder{}) }

// PreOrder warns when a shop's page takes pre-orders and the draft lacks the
// availability PreOrder or a PreOrderAction for the order form.
type PreOrder struct{}

// ID is the check's ID.
func (PreOrder) ID() string { return "product/pre-order" }

// Run reports the first mention of pre-orders.
func (PreOrder) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() {
		return nil
	}
	m := webpage.PreOrder.FindString(c.Page.Main)
	if m == "" || (jsonld.KeyMember(c.Nodes, "availability", "PreOrder") && c.Has("PreOrderAction")) {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page takes pre-orders (%q): set the Offer's availability https://schema.org/PreOrder and add a PreOrderAction as its potentialAction (target the order form), with availabilityStarts when the page gives a date", m)}
}
