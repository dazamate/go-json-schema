package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestLowStock(t *testing.T) {
	page := shopPageWith("<p>Only 3 left in stock.</p>")
	level := `{"@context":"https://schema.org","@type":"Offer","inventoryLevel":{"@type":"QuantitativeValue","value":3}}`
	limited := `{"@context":"https://schema.org","@type":"Offer","availability":"https://schema.org/LimitedAvailability"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a low stock level", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Only 3 left in stock": set each such product's Offer inventoryLevel (a QuantitativeValue with the number left as value) and availability https://schema.org/LimitedAvailability`},
		}},
		{"an inventoryLevel holds it", audittest.Input{Draft: level, PageHTML: page, PageURL: notesURL}, nil},
		{"LimitedAvailability is there", audittest.Input{Draft: limited, PageHTML: page, PageURL: notesURL}, nil},
		{"no stock level", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.LowStock{}, tc.in, tc.want) })
	}
}
