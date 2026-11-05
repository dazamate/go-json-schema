package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestTransitTime(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"ShippingConditions","transitTime":{"@type":"QuantitativeValue","minValue":2,"maxValue":6,"unitCode":"DAY"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a delivery time", audittest.Input{Draft: noDraft, PageHTML: shippingPage, PageURL: shippingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Delivery in 2-6 business days" but no transitTime carries it: set transitTime on the ShippingConditions (a ServicePeriod with businessDays and a duration of 2 to 6 days) or in the Offer's ShippingDeliveryTime (a QuantitativeValue with minValue 2, maxValue 6 and unitCode DAY)`},
		}},
		{"a transitTime carries it", audittest.Input{Draft: carried, PageHTML: shippingPage, PageURL: shippingURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(shippingPage), PageURL: shippingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: noindex(shippingPage), PageURL: shippingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: shippingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.TransitTime{}, tc.in, tc.want) })
	}
}
