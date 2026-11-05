package policies

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ReturnPolicy{}) }

// ReturnPolicy warns when a shop's page states its return policy, or links
// the policy's page, and the draft has no MerchantReturnPolicy: Google reads
// a shop's returns from that node, on the business and on each Offer.
type ReturnPolicy struct{}

// ID is the check's ID.
func (ReturnPolicy) ID() string { return "policies/return-policy" }

// Run reports the terms the page states, or its link to the policy, once,
// when the draft has no MerchantReturnPolicy.
func (ReturnPolicy) Run(c *audit.Context) []audit.Issue {
	r := returnsOf(c)
	if r == nil || len(r.policies) > 0 || (len(r.facts) == 0 && r.link == "") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page %s that no MerchantReturnPolicy carries: give the business (and each Offer) hasMerchantReturnPolicy, a MerchantReturnPolicy with applicableCountry, %s", r.what(), strings.Join(r.set(), ", "))}
}
