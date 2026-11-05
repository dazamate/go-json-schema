package policies_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestPayments(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"OnlineStore","acceptedPaymentMethod":"https://schema.org/PayPal"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"cards, a wallet and a transfer", audittest.Input{Draft: noDraft, PageHTML: paymentsPage, PageURL: paymentsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows how to pay (Visa, Mastercard, PayPal, https://schema.org/ByBankTransferInAdvance): set acceptedPaymentMethod on the business: a PaymentCard for each card (Visa, Mastercard); a PaymentMethod for each other way to pay (PayPal); a PaymentMethod with paymentMethodType https://schema.org/ByBankTransferInAdvance (a PaymentMethodType)"},
		}},
		{"a payment logo", audittest.Input{Draft: noDraft, PageHTML: logoPaymentsPage, PageURL: paymentsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows how to pay (Klarna): set acceptedPaymentMethod on the business: a PaymentMethod for each other way to pay (Klarna)"},
		}},
		{"discover as a word", audittest.Input{Draft: noDraft, PageHTML: payLine("Discover our new range of rings. A visa is not needed."), PageURL: paymentsURL}, nil},
		{"a discover card", audittest.Input{Draft: noDraft, PageHTML: payLine("We take Discover card."), PageURL: paymentsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows how to pay (Discover): set acceptedPaymentMethod on the business: a PaymentCard for each card (Discover)"},
		}},
		{"visa accepted alone", audittest.Input{Draft: noDraft, PageHTML: payLine("We accept Visa. You can also pay with Discover."), PageURL: paymentsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows how to pay (Visa, Discover): set acceptedPaymentMethod on the business: a PaymentCard for each card (Visa, Discover)"},
		}},
		{"discover in a list", audittest.Input{Draft: noDraft, PageHTML: payLine("We accept Mastercard and Discover."), PageURL: paymentsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows how to pay (Mastercard, Discover): set acceptedPaymentMethod on the business: a PaymentCard for each card (Mastercard, Discover)"},
		}},
		{"a discover icon", audittest.Input{Draft: noDraft, PageHTML: payLine(`<img src="/d.svg" alt="Discover">`), PageURL: paymentsURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page shows how to pay (Discover): set acceptedPaymentMethod on the business: a PaymentCard for each card (Discover)"},
		}},
		{"a discover link", audittest.Input{Draft: noDraft, PageHTML: payLine(`<a href="/new" title="Discover more">New</a>`), PageURL: paymentsURL}, nil},
		{"acceptedPaymentMethod is there", audittest.Input{Draft: carried, PageHTML: paymentsPage, PageURL: paymentsURL}, nil},
		{"a page that names no way to pay", audittest.Input{Draft: noDraft, PageHTML: returnsPage, PageURL: returnsURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(paymentsPage), PageURL: paymentsURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(paymentsPage), PageURL: paymentsURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: paymentsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.Payments{}, tc.in, tc.want) })
	}
}

// logoPaymentsPage shows a payment method only as a logo.
const logoPaymentsPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Harbour Jewellers</h1></main><footer><img src="/klarna.svg" alt="Klarna"></footer></body></html>`

// payLine is the payments page with the line for its way-to-pay text.
func payLine(line string) string {
	return strings.Replace(paymentsPage, "We accept Visa, Mastercard and PayPal. Pay by bank transfer if you prefer.", line, 1)
}
