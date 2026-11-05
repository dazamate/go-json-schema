package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestSaleEnds(t *testing.T) {
	page := shopPageWith("<p>Sale ends 30 November 2026.</p>")
	carried := `{"@context":"https://schema.org","@type":"Offer","priceValidUntil":"2026-11-30"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a sale with an end", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Sale ends 30 November 2026": set priceValidUntil 2026-11-30 on the sale Offer; a store-wide sale can also be a SaleEvent (name, startDate, endDate 2026-11-30)`},
		}},
		{"priceValidUntil is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"no end date", audittest.Input{Draft: noDraft, PageHTML: shopPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a page that sells nothing", audittest.Input{Draft: noDraft, PageHTML: sellingNoMore(page), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.SaleEnds{}, tc.in, tc.want) })
	}
}
