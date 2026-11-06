package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestReturnLink(t *testing.T) {
	policy := `{"@context":"https://schema.org","@type":"MerchantReturnPolicy","returnPolicyCategory":"https://schema.org/MerchantReturnFiniteReturnWindow"}`
	linked := `{"@context":"https://schema.org","@type":"MerchantReturnPolicy","returnPolicyCategory":"https://schema.org/MerchantReturnFiniteReturnWindow","merchantReturnLink":"https://harbour.test/pages/returns"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"the page links the policy", audittest.Input{Draft: policy, PageHTML: footerReturnsPage, PageURL: homeURL}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "the page links the return policy (https://harbour.test/pages/returns): set merchantReturnLink https://harbour.test/pages/returns on the MerchantReturnPolicy"},
		}},
		{"the policy page itself", audittest.Input{Draft: policy, PageHTML: returnsPage, PageURL: returnsURL}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "the page links the return policy (https://harbour.test/pages/returns): set merchantReturnLink https://harbour.test/pages/returns on the MerchantReturnPolicy"},
		}},
		{"the policy has the link", audittest.Input{Draft: linked, PageHTML: footerReturnsPage, PageURL: homeURL}, nil},
		{"no MerchantReturnPolicy", audittest.Input{Draft: noDraft, PageHTML: footerReturnsPage, PageURL: homeURL}, nil},
		{"no link to a policy", audittest.Input{Draft: policy, PageHTML: paymentsPage, PageURL: paymentsURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: policy, PageHTML: sellingNoMore(footerReturnsPage), PageURL: homeURL}, nil},
		{"a noindex page", audittest.Input{Draft: policy, PageHTML: audittest.Noindex(footerReturnsPage), PageURL: homeURL}, nil},
		{"no page", audittest.Input{Draft: policy, PageURL: returnsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.ReturnLink{}, tc.in, tc.want) })
	}
}
