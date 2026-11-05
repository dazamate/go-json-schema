package policies_test

import (
	"fmt"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

// The return window is worded many ways.
func TestReturnWindowWordings(t *testing.T) {
	category := `{"@context":"https://schema.org","@type":"MerchantReturnPolicy","returnPolicyCategory":"https://schema.org/MerchantReturnFiniteReturnWindow"}`
	for _, tc := range []struct {
		name, line, said, days string
		also                   []audit.Issue
	}{
		// royalcanadiancollective.com FAQ
		{"returns accepted within", "Returns accepted within 30 days of purchase. Items must be unworn.", "Returns accepted within 30 days", "30", nil},
		// oliverbonas.com returns page
		{"return most items to us within", "If you change your mind, you can return most items to us within 30 days of purchase or receiving your order.", "return most items to us within 30 days", "30", nil},
		// kasaustralia.com banner
		{"days free return", "Free shipping on orders over $150 30 days free return", "30 days free return", "30", []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `the page says "free return" but this MerchantReturnPolicy does not: set returnFees https://schema.org/FreeReturn`},
		}},
		{"weeks", "You may return items within 4 weeks of delivery.", "return items within 4 weeks", "28", nil},
		{"a month", "Return unused items within 1 month.", "Return unused items within 1 month", "30", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header><main><h1>Returns</h1><p>` + tc.line + `</p></main></body></html>`
			want := append([]audit.Issue{{Path: "$", Severity: audit.Warning, Message: fmt.Sprintf(`the page says %q but this MerchantReturnPolicy does not: set merchantReturnDays %s`, tc.said, tc.days)}}, tc.also...)
			audittest.Expect(t, policies.ReturnTerm{}, audittest.Input{Draft: category, PageHTML: page, PageURL: returnsURL}, want)
		})
	}
}
