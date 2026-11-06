package policies_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

// cartLink is the link that makes a page a shop's.
const cartLink = `<nav><a href="/cart">Cart</a></nav>`

// shippingPage is a shop's shipping policy: a flat rate, a free-shipping
// threshold, a handling time and a transit time.
const shippingPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Shipping</h1>
<p>Shipping within Australia is a flat rate of $7.95. Free shipping on orders over $60.</p>
<p>Orders are dispatched within 1-2 business days. Delivery in 2-6 business days.</p></main></body></html>`

const (
	shippingURL = "https://harbour.test/pages/shipping"
	noDraft     = `{"@context":"https://schema.org","@graph":[]}`
)

// sellingNoMore is the page without its cart link.
func sellingNoMore(page string) string { return strings.Replace(page, cartLink, "", 1) }

func TestShippingRate(t *testing.T) {
	// "Free over $60, otherwise $7.95" gives the rate after the threshold
	banner := `<!doctype html><html><head></head><body><header>` + cartLink + `</header><main><h1>Harbour Jewellers</h1><p>Free over $60, otherwise $7.95.</p></main></body></html>`
	carried := `{"@context":"https://schema.org","@type":"ShippingService","shippingConditions":{"@type":"ShippingConditions","shippingRate":{"@type":"MonetaryAmount","value":7.95,"currency":"AUD"}}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a flat rate within a country", audittest.Input{Draft: noDraft, PageHTML: shippingPage, PageURL: shippingURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page gives a shipping rate of $7.95 that no node carries: add a ShippingService to the business's hasShippingService (fulfillmentType https://schema.org/FulfillmentTypeDelivery) with shippingConditions, a ShippingConditions with shippingDestination (a DefinedRegion with addressCountry AU) and shippingRate (a MonetaryAmount of 7.95); on a product page give its Offer the same as shippingDetails, an OfferShippingDetails"},
		}},
		{"free over a threshold, otherwise a rate", audittest.Input{Draft: noDraft, PageHTML: banner, PageURL: "https://harbour.test/"}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page gives a shipping rate of $7.95 that no node carries: add a ShippingService to the business's hasShippingService (fulfillmentType https://schema.org/FulfillmentTypeDelivery) with shippingConditions, a ShippingConditions with shippingDestination (a DefinedRegion with addressCountry its country) and shippingRate (a MonetaryAmount of 7.95); on a product page give its Offer the same as shippingDetails, an OfferShippingDetails"},
		}},
		{"a shipping node carries it", audittest.Input{Draft: carried, PageHTML: shippingPage, PageURL: shippingURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(shippingPage), PageURL: shippingURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(shippingPage), PageURL: shippingURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: shippingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.ShippingRate{}, tc.in, tc.want) })
	}
}
