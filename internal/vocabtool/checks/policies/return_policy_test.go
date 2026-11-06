package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestReturnPolicy(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"MerchantReturnPolicy","returnPolicyCategory":"https://schema.org/MerchantReturnFiniteReturnWindow"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"terms the page states", audittest.Input{Draft: noDraft, PageHTML: returnsPage, PageURL: returnsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page states a return policy ("Return any item within 14 days", "full refund") that no MerchantReturnPolicy carries: give the business (and each Offer) hasMerchantReturnPolicy, a MerchantReturnPolicy with applicableCountry, merchantReturnDays 14, returnPolicyCategory https://schema.org/MerchantReturnFiniteReturnWindow, refundType https://schema.org/FullRefund, merchantReturnLink https://harbour.test/pages/returns`},
		}},
		{"a link to the policy", audittest.Input{Draft: noDraft, PageHTML: footerReturnsPage, PageURL: homeURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page links its return policy that no MerchantReturnPolicy carries: give the business (and each Offer) hasMerchantReturnPolicy, a MerchantReturnPolicy with applicableCountry, merchantReturnLink https://harbour.test/pages/returns`},
		}},
		{"a MerchantReturnPolicy is there", audittest.Input{Draft: carried, PageHTML: returnsPage, PageURL: returnsURL}, nil},
		{"a page that states and links nothing", audittest.Input{Draft: noDraft, PageHTML: paymentsPage, PageURL: paymentsURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(returnsPage), PageURL: returnsURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(returnsPage), PageURL: returnsURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: "Return any item within 14 days for a full refund.", PageURL: returnsURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: returnsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.ReturnPolicy{}, tc.in, tc.want) })
	}
}
