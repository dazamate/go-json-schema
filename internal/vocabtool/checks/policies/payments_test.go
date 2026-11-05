package policies_test

import (
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
