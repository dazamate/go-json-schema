package policies_test

import (
	"fmt"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

// shopLine is a shop's page that says one thing about its shipping.
func shopLine(line string) string {
	return `<!doctype html><html><head></head><body><header><a href="/">Harbour Boots</a>` + cartLink + `</header>
<main><h1>Shipping</h1><p>` + line + `</p></main></body></html>`
}

func warn(msg string, args ...any) []audit.Issue {
	return []audit.Issue{{Path: "html:body", Severity: audit.Warning, Message: fmt.Sprintf(msg, args...)}}
}

const freeMsg = "the page offers free shipping over %s%s but no ShippingConditions has an orderValue of %s: add a ShippingConditions with orderValue (a MonetaryAmount with minValue %s) and shippingRate 0 to the business's ShippingService, or a ShippingRateSettings whose freeShippingThreshold is a DeliveryChargeSpecification of %s"

// Free shipping is worded many ways, in many currencies.
func TestFreeShippingWordings(t *testing.T) {
	for _, tc := range []struct {
		name, line, sym, amount string
	}{
		// royalcanadiancollective.com FAQ
		{"shipping free on orders and up", "Is shipping free? Shipping free on orders $34.99 and up; orders under $34.99 ship $9.", "$", "34.99"},
		// totallyirishgifts.com banner
		{"to a country, in a currency code", "Free Shipping to Ireland & Northern Ireland on Orders Over EUR75 | EU Delivery Available", "€", "75"},
		// oliverbonas.com banner
		{"pounds, in capitals", "FREE CLICK & COLLECT OVER £30 | FREE UK STANDARD DELIVERY OVER £60 (EXC FURNITURE)", "£", "60"},
		// kasaustralia.com banner
		{"on orders, with no over", "Free shipping on orders $150 30 days free return", "$", "150"},
		{"a dotted country", "Free U.S. shipping on orders over $1,000.", "$", "1,000"},
		{"when you spend over", "Free delivery when you spend over £45.", "£", "45"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, policies.FreeShipping{}, audittest.Input{Draft: noDraft, PageHTML: shopLine(tc.line), PageURL: shippingURL}, warn(freeMsg, tc.sym, tc.amount, tc.amount, tc.amount, tc.amount))
		})
	}
	// royalcanadiancollective.com FAQ: free on every order
	t.Run("free shipping with no threshold", func(t *testing.T) {
		audittest.Expect(t, policies.FreeShipping{}, audittest.Input{Draft: noDraft, PageHTML: shopLine("Free shipping applies to all boot and slipper orders."), PageURL: shippingURL}, nil)
	})
	// taylorstitch.com shipping policy
	t.Run("no free shipping offered", func(t *testing.T) {
		audittest.Expect(t, policies.FreeShipping{}, audittest.Input{Draft: noDraft, PageHTML: shopLine("There is no free shipping available on international orders. We require signature upon delivery on orders over $750."), PageURL: shippingURL}, nil)
	})
}

const rateMsg = "the page gives a shipping rate of %s%s that no node carries: add a ShippingService to the business's hasShippingService (fulfillmentType https://schema.org/FulfillmentTypeDelivery) with shippingConditions, a ShippingConditions with shippingDestination (a DefinedRegion with addressCountry %s) and shippingRate (a MonetaryAmount of %s); on a product page give its Offer the same as shippingDetails, an OfferShippingDetails"

func TestShippingRateWordings(t *testing.T) {
	for _, tc := range []struct {
		name, line, sym, amount, country string
	}{
		// royalcanadiancollective.com FAQ
		{"orders under a sum ship for", "Orders under $34.99 ship $9.", "$", "9", "its country"},
		{"a delivery charge in pounds", "We deliver orders to the UK. Standard delivery charge £3.95.", "£", "3.95", "GB"},
		{"shipping and handling", "Shipping & handling $10 per order.", "$", "10", "its country"},
		{"delivery costs in euro", "Orders to Ireland. Delivery costs: €4.50.", "€", "4.50", "IE"},
		{"shipping from", "Shipping from $5 for orders across Canada.", "$", "5", "CA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, policies.ShippingRate{}, audittest.Input{Draft: noDraft, PageHTML: shopLine(tc.line), PageURL: shippingURL}, warn(rateMsg, tc.sym, tc.amount, tc.country, tc.amount))
		})
	}
}

func TestTransitTimeWordings(t *testing.T) {
	const msg = "the page says %q but no transitTime carries it: set transitTime on the ShippingConditions (a ServicePeriod with businessDays and a duration of %s) or in the Offer's ShippingDeliveryTime (a QuantitativeValue with minValue %s, maxValue %s and unitCode DAY)"
	for _, tc := range []struct {
		name, line, said, span, lo, hi string
	}{
		// taylorstitch.com shipping policy: the range wins over the single number
		{"allow a range for arrival", "Once you receive your tracking number via email, most orders arrive within 3 business days, but please allow 5-10 business days for shipment arrival depending on carrier factors.", "allow 5-10 business days for shipment", "5 to 10 days", "5", "10"},
		{"arrive within one number", "Most orders arrive within 3 business days.", "arrive within 3 business days", "3 days", "3", "3"},
		{"delivery takes", "Delivery takes 3-5 working days.", "Delivery takes 3-5 working days", "3 to 5 days", "3", "5"},
		{"delivered within, in weeks", "Delivered within 1-2 weeks.", "Delivered within 1-2 weeks", "7 to 14 days", "7", "14"},
		// royalcanadiancollective.com FAQ
		{"arrive within a range", "Most deliveries arrive within 2 to 7 business days.", "arrive within 2 to 7 business days", "2 to 7 days", "2", "7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, policies.TransitTime{}, audittest.Input{Draft: noDraft, PageHTML: shopLine(tc.line), PageURL: shippingURL}, warn(msg, tc.said, tc.span, tc.lo, tc.hi))
		})
	}
	t.Run("a return's delivery", func(t *testing.T) {
		audittest.Expect(t, policies.TransitTime{}, audittest.Input{Draft: noDraft, PageHTML: shopLine("Please allow 1-3 business days from the time your return is delivered to our warehouse for us to process it."), PageURL: shippingURL}, nil)
	})
}

func TestHandlingTimeWordings(t *testing.T) {
	const msg = "the page says %s but no handlingTime carries it: set handlingTime on the ShippingService (a ServicePeriod, or a QuantitativeValue with maxValue %d and unitCode DAY) or, on a product page, in its OfferShippingDetails's deliveryTime (a ShippingDeliveryTime with handlingTime a QuantitativeValue)"
	for _, tc := range []struct {
		name, line, said string
		days             int
	}{
		// totallyirishgifts.com product page
		{"typically dispatched in", "Prepared by maker: typically dispatched in 3–6 business days, based on recent orders.", `"dispatched in 3–6 business days"`, 6},
		// taylorstitch.com shipping policy
		{"allow up to to fulfill", "For in-stock items, please allow up to 2 business days to fulfill an order and issue a tracking number.", `"allow up to 2 business days to fulfill"`, 2},
		{"ships in, British spelling", "Orders despatched within 24 hours.", `"despatched within 24 hours"`, 1},
		{"processing time is, in weeks", "Processing time is 2 weeks.", `"Processing time is 2 weeks"`, 14},
		{"takes to process", "Each order takes 1-2 days to process.", `"takes 1-2 days to process"`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, policies.HandlingTime{}, audittest.Input{Draft: noDraft, PageHTML: shopLine(tc.line), PageURL: shippingURL}, warn(msg, tc.said, tc.days))
		})
	}
	// a refund is processed in days, not an order
	t.Run("a refund processed in days", func(t *testing.T) {
		audittest.Expect(t, policies.HandlingTime{}, audittest.Input{Draft: noDraft, PageHTML: shopLine("Your refund is processed in 3-5 business days."), PageURL: shippingURL}, nil)
	})
}
