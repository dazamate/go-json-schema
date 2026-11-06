package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestFoundingDate(t *testing.T) {
	page := plainPageWith("<p>Founded in 2012 by Maria Santos.</p>")
	carried := `{"@context":"https://schema.org","@type":"Organization","foundingDate":"2012"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a founding year", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: notesURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "Founded in 2012": set the business's foundingDate 2012`},
		}},
		{"foundingDate is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: notesURL}, nil},
		{"no founding year", audittest.Input{Draft: noDraft, PageHTML: plainPageWith("<p>Handmade rings.</p>"), PageURL: notesURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: notesURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: notesURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.FoundingDate{}, tc.in, tc.want) })
	}
}
