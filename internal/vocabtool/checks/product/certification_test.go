package product_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
)

func TestCertification(t *testing.T) {
	page := productPageWith("Art Deco ring", "Harbour Jewellers", "<p>Each diamond is GIA certified.</p>")
	carried := `{"@context":"https://schema.org","@type":"Certification","name":"GIA"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a certificate", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: productURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page says "GIA certified": set the Product's hasCertification, a Certification (name, issuedBy the certifying body, certificationIdentification)`},
		}},
		{"a Certification is there", audittest.Input{Draft: carried, PageHTML: page, PageURL: productURL}, nil},
		{"no certificate", audittest.Input{Draft: noDraft, PageHTML: productPageWith("Art Deco ring", "Harbour Jewellers", ""), PageURL: productURL}, nil},
		{"a page that is no product page", audittest.Input{Draft: noDraft, PageHTML: page, PageURL: otherURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(page), PageURL: productURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: productURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, product.Certification{}, tc.in, tc.want) })
	}
}
