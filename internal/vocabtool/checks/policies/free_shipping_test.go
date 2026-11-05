package policies_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestFreeShipping(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"ShippingConditions","orderValue":{"@type":"MonetaryAmount","minValue":60,"currency":"AUD"},"shippingRate":{"@type":"MonetaryAmount","value":0,"currency":"AUD"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"free shipping over a threshold", audittest.Input{Draft: noDraft, PageHTML: shippingPage, PageURL: shippingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page offers free shipping over $60 but no ShippingConditions has an orderValue of 60: add a ShippingConditions with orderValue (a MonetaryAmount with minValue 60) and shippingRate 0 to the business's ShippingService, or a ShippingRateSettings whose freeShippingThreshold is a DeliveryChargeSpecification of 60"},
		}},
		{"a threshold in pounds", audittest.Input{Draft: noDraft, PageHTML: strings.NewReplacer("$", "£").Replace(shippingPage), PageURL: shippingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page offers free shipping over £60 but no ShippingConditions has an orderValue of 60: add a ShippingConditions with orderValue (a MonetaryAmount with minValue 60) and shippingRate 0 to the business's ShippingService, or a ShippingRateSettings whose freeShippingThreshold is a DeliveryChargeSpecification of 60"},
		}},
		{"a ShippingConditions carries it", audittest.Input{Draft: carried, PageHTML: shippingPage, PageURL: shippingURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(shippingPage), PageURL: shippingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(shippingPage), PageURL: shippingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: shippingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.FreeShipping{}, tc.in, tc.want) })
	}
}
