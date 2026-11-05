package shop

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(BusinessType{}) }

var _ audit.Retyper = BusinessType{}

// BusinessType warns when a shop's business node is typed more generally
// than the shop: one that sells online is an OnlineStore, and one with a
// street address the Store subtype for its trade. It leaves be a node an
// earlier source in the retype ledger told. It needs the page's document, on
// a page search engines index.
type BusinessType struct{}

// ID is the check's ID.
func (BusinessType) ID() string { return "shop/business-type" }

// find is the shop's business and its node's type, when the shop calls for a
// subtype of that type and no source before it told the node.
func (BusinessType) find(c *audit.Context, before audit.Retyped) (b *shopBusiness, ty string, ok bool) {
	b = shopBusinessOf(c)
	if b == nil || b.node.N == nil {
		return nil, "", false
	}
	ty = c.Single(b.node.N)
	if ty == "" || ty == b.want || !slices.Contains(c.Vocab.V.Ancestors(b.want), ty) || before.Told[b.node.N] {
		return nil, "", false
	}
	return b, ty, true
}

// Run reports the business node with the type the shop takes.
func (k BusinessType) Run(c *audit.Context) []audit.Issue {
	b, ty, ok := k.find(c, c.RetypedBefore(k.ID()))
	if !ok {
		return nil
	}
	return []audit.Issue{audit.Issuef(b.node.Path, audit.Warning, "%s: type this %s %s", b.why, ty, b.want)}
}

// Retypes tells the business node and names the type the shop takes.
func (k BusinessType) Retypes(c *audit.Context, before audit.Retyped) []audit.Retype {
	b, _, ok := k.find(c, before)
	if !ok {
		return nil
	}
	return []audit.Retype{{Node: b.node.N, Named: []string{b.want}}}
}
