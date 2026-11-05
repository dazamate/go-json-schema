package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

// royalcanadiancollective.com says "proudly Canadian owned and operated since
// 2012", without "founded".
func TestFoundingDateSince(t *testing.T) {
	audittest.Expect(t, product.FoundingDate{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: plainPageWith(`<p>We have been proudly Canadian owned and operated since 2012, and our boots are designed in Canada.</p>`)}, []audit.Issue{
		{Path: "html:body", Severity: audit.Warning, Message: `the page says "owned and operated since 2012": set the business's foundingDate 2012`},
	})
	audittest.Expect(t, product.FoundingDate{}, audittest.Input{Draft: noDraft, PageURL: notesURL,
		PageHTML: plainPageWith(`<p>Since 2020 house prices have risen. As of July 2026, homes sold below list price.</p>`)}, nil)
}
