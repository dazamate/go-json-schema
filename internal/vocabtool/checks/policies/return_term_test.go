package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestReturnTerm(t *testing.T) {
	category := `{"@context":"https://schema.org","@type":"MerchantReturnPolicy","returnPolicyCategory":"https://schema.org/MerchantReturnFiniteReturnWindow"}`
	full := `{"@context":"https://schema.org","@type":"MerchantReturnPolicy","merchantReturnDays":14,"returnPolicyCategory":"https://schema.org/MerchantReturnFiniteReturnWindow","refundType":"https://schema.org/FullRefund"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"terms the policy leaves out", audittest.Input{Draft: category, PageHTML: returnsPage, PageURL: returnsURL}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `the page says "Return any item within 14 days" but this MerchantReturnPolicy does not: set merchantReturnDays 14`},
			{Path: "$", Severity: audit.Warning, Message: `the page says "full refund" but this MerchantReturnPolicy does not: set refundType https://schema.org/FullRefund`},
		}},
		{"the policy carries every term", audittest.Input{Draft: full, PageHTML: returnsPage, PageURL: returnsURL}, nil},
		{"no MerchantReturnPolicy", audittest.Input{Draft: noDraft, PageHTML: returnsPage, PageURL: returnsURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: category, PageHTML: sellingNoMore(returnsPage), PageURL: returnsURL}, nil},
		{"a noindex page", audittest.Input{Draft: category, PageHTML: audittest.Noindex(returnsPage), PageURL: returnsURL}, nil},
		{"no page", audittest.Input{Draft: category, PageURL: returnsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.ReturnTerm{}, tc.in, tc.want) })
	}
}
