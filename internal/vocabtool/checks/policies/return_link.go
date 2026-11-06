package policies

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(ReturnLink{}) }

// ReturnLink warns when a shop's page links its return policy and no
// MerchantReturnPolicy names that page as its merchantReturnLink: the link is
// how Google finds the policy's full text.
type ReturnLink struct{}

// ID is the check's ID.
func (ReturnLink) ID() string { return "policies/return-link" }

// Run reports the link on the first MerchantReturnPolicy, when none of them
// carries it.
func (ReturnLink) Run(c *audit.Context) []audit.Issue {
	r := returnsOf(c)
	if r == nil || len(r.policies) == 0 || r.link == "" {
		return nil
	}
	if slices.ContainsFunc(r.policies, func(l jsonld.Node) bool {
		return slices.ContainsFunc(jsonld.Strs(l.N, "merchantReturnLink"), func(s string) bool { return txt.NormLink(s) == r.link })
	}) {
		return nil
	}
	return []audit.Issue{audit.Issuef(r.policies[0].Path, audit.Warning, "the page links the return policy (%s): set merchantReturnLink %s on the MerchantReturnPolicy", r.link, r.link)}
}
