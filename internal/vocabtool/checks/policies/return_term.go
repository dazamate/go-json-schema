package policies

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(ReturnTerm{}) }

// ReturnTerm warns on each term of the return policy a shop's page states
// (a return window, who pays the postage, the refund type) that no
// MerchantReturnPolicy in the draft carries: a policy that leaves out what
// the page promises tells Google something else.
type ReturnTerm struct{}

// ID is the check's ID.
func (ReturnTerm) ID() string { return "policies/return-term" }

// Run reports each term none of the draft's MerchantReturnPolicy nodes
// carries, on the first of them.
func (ReturnTerm) Run(c *audit.Context) []audit.Issue {
	r := returnsOf(c)
	if r == nil || len(r.policies) == 0 {
		return nil
	}
	var out []audit.Issue
	for _, f := range r.facts {
		ok := slices.ContainsFunc(r.policies, func(l jsonld.Node) bool {
			if f.prop == "returnPolicySeasonalOverride" {
				_, has := l.N.Get(f.prop)
				return has
			}
			if f.prop == "merchantReturnDays" {
				return slices.Contains(jsonld.Scalars(l.N, f.prop), f.member)
			}
			return slices.ContainsFunc(jsonld.Strs(l.N, f.prop), func(s string) bool { return jsonld.StripPrefix(s) == f.member })
		})
		if !ok {
			out = append(out, audit.Issuef(r.policies[0].Path, audit.Warning, "the page says %q but this MerchantReturnPolicy does not: set %s", f.said, f.describe()))
		}
	}
	return out
}
