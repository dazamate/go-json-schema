package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestHandlingTime(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"ShippingService","handlingTime":{"@type":"QuantitativeValue","minValue":1,"maxValue":2,"unitCode":"DAY"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a dispatch time", audittest.Input{Draft: noDraft, PageHTML: shippingPage, PageURL: shippingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "dispatched within 1-2 business days" but no handlingTime carries it: set handlingTime on the ShippingService (a ServicePeriod, or a QuantitativeValue with maxValue 2 and unitCode DAY) or, on a product page, in its OfferShippingDetails's deliveryTime (a ShippingDeliveryTime with handlingTime a QuantitativeValue)`},
		}},
		{"a handlingTime carries it", audittest.Input{Draft: carried, PageHTML: shippingPage, PageURL: shippingURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(shippingPage), PageURL: shippingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(shippingPage), PageURL: shippingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: shippingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.HandlingTime{}, tc.in, tc.want) })
	}
}
