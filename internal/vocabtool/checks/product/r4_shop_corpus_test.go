package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// maxum.com.au: a watch's engraving is a paid add-on in a control's title, in
// capitals, with a plus sign before the price.
func TestAddOnCustomEngraving(t *testing.T) {
	audittest.Expect(t, product.AddOn{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Quantity Add to cart ✦ ADD CUSTOM ENGRAVING +$20 ▾ Add your message 0 /25</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page offers "ADD CUSTOM ENGRAVING +$20" as an extra: add it as the Offer's addOn, an Offer (name, price, priceCurrency)`},
	})
	// a warranty or protection plan is a paid extra the same way
	audittest.Expect(t, product.AddOn{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Add 2-year protection plan for £49.00</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page offers "Add 2-year protection plan for £49.00" as an extra: add it as the Offer's addOn, an Offer (name, price, priceCurrency)`},
	})
	// "add your message" is no priced extra
	audittest.Expect(t, product.AddOn{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Add your message to the card. Postage $9</p>`)}, nil)
}

// asda.com and hellosupermarket.co.uk: a UK grocer's unit price in brackets
// after the price, and a price per 100g in pence.
func TestUnitPriceGrocer(t *testing.T) {
	audittest.Expect(t, product.UnitPrice{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Trophy Basmati Rice 10kg 10kg actual price £13.00 (£1.30/kg) Add</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page gives a unit price ("£1.30/kg"): add a UnitPriceSpecification with that price and referenceQuantity (a QuantitativeValue with value and unitCode) to the Offer's priceSpecification`},
	})
	audittest.Expect(t, product.UnitPrice{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Tesco Macaroni Pasta 1kg £1.19 Seen 5 Nov 2026 12p per 100g 1kg</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page gives a unit price ("12p per 100g"): add a UnitPriceSpecification with that price and referenceQuantity (a QuantitativeValue with value and unitCode) to the Offer's priceSpecification`},
	})
	// a US price per ounce is one
	audittest.Expect(t, product.UnitPrice{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Numi Organic Tea $5.99 ($0.45/oz)</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page gives a unit price ("$0.45/oz"): add a UnitPriceSpecification with that price and referenceQuantity (a QuantitativeValue with value and unitCode) to the Offer's priceSpecification`},
	})
	// a Shopify theme's empty unit-price label is no unit price
	audittest.Expect(t, product.UnitPrice{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Regular price From $35.90 NZD Unit price / per</p>`)}, nil)
}

// tangelos.co.uk, honest tea: certifications by body.
func TestCertificationBodies(t *testing.T) {
	for _, tc := range []struct{ body, said string }{
		{"<p>Free UK Delivery Certified Organic GOTS-certified organic cotton, from seed to stitch.</p>", "Certified Organic GOTS"},
		{"<p>Real-brewed with Fair Trade Certified tea leaves and cane sugar.</p>", "Fair Trade Certified"},
		{"<p>This kettle is UL listed and CE marked.</p>", "UL listed"},
		{"<p>An Energy Star certified washer.</p>", "Energy Star certified"},
	} {
		audittest.Expect(t, product.Certification{}, audittest.Input{Draft: noDraft, PageURL: productURL,
			PageHTML: productPageWith("Item", "Harbour", tc.body)}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "` + tc.said + `": set the Product's hasCertification, a Certification (name, issuedBy the certifying body, certificationIdentification)`},
		})
	}
	// organic coffee's name is no certificate statement
	audittest.Expect(t, product.Certification{}, audittest.Input{Draft: noDraft, PageURL: productURL,
		PageHTML: productPageWith("Organic Blend", "Harbour", "<p>The New Zealand Coffee Co Fairtrade Organic. Organic and bursting with flavor!</p>")}, nil)
}

// vicisouthbank.com.au: a weekend or public-holiday surcharge is no card
// surcharge.
func TestSurchargeNotCard(t *testing.T) {
	audittest.Expect(t, product.Surcharge{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Additional Info 10% surcharge applies on weekends | 20% surcharge applies on public holidays</p>`)}, nil)
	audittest.Expect(t, product.Surcharge{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: shopPageWith(`<p>Weekend surcharges apply. A 1.5% surcharge applies to credit card payments.</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page says "1.5% surcharge": add a PaymentChargeSpecification (appliesToPaymentMethod the cards it applies to, and the charge) to the Offer's priceSpecification`},
	})
}
