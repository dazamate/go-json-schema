package shop

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(LocalStore{}) }

// LocalStore warns when a shop's page shows a street address and the draft
// has no business node: a shop with premises is a LocalBusiness, typed as the
// Store subtype for its trade, with its address, location and hours. It
// needs the page's document, on a page search engines index.
type LocalStore struct{}

// ID is the check's ID.
func (LocalStore) ID() string { return "shop/local-store" }

// Run reports a shop with a street address and no business node.
func (LocalStore) Run(c *audit.Context) []audit.Issue {
	b := shopBusinessOf(c)
	if b == nil || b.node.N != nil || b.want == "OnlineStore" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "%s but no business node: add the business as a %s (the Store subtype for its trade, a LocalBusiness: name, address, geo, openingHoursSpecification, telephone)", b.why, b.want)}
}
