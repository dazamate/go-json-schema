package shop

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(OnlineStore{}) }

// OnlineStore warns when a shop that sells online and shows no street
// address has no business node: the business that runs the site is an
// OnlineStore, and the WebSite's publisher. It needs the page's document, on
// a page search engines index.
type OnlineStore struct{}

// ID is the check's ID.
func (OnlineStore) ID() string { return "shop/online-store" }

// Run reports a shop that sells online with no business node.
func (OnlineStore) Run(c *audit.Context) []audit.Issue {
	b := shopBusinessOf(c)
	if b == nil || b.node.N != nil || b.want != "OnlineStore" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the site sells online but the page has no business node: add an OnlineStore (the OnlineBusiness subtype of Organization for a shop that trades online) as the WebSite's publisher, with name, url, logo, sameAs, email, taxID, hasMerchantReturnPolicy and hasShippingService")}
}
