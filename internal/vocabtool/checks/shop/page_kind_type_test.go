package shop_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

func TestPageKindType(t *testing.T) {
	// "Get in touch" names no subtype, so only the page's kind types it
	plain := `{"@context":"https://schema.org","@type":"WebPage","name":"Get in touch"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a contact page's WebPage", audittest.Input{Draft: plain, PageHTML: contactPage, PageURL: contactURL}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "this WebPage is a contact page: type it ContactPage"},
		}},
		{"typed for its kind", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"ContactPage","name":"Get in touch"}`, PageHTML: contactPage, PageURL: contactURL}, nil},
		// medicalAudience is a MedicalWebPage's: the subtype check told it first
		{"a node another signal told", audittest.Input{Draft: `{"@context":"https://schema.org","@type":"WebPage","name":"Get in touch","medicalAudience":{"@type":"MedicalAudience","name":"Patients"}}`, PageHTML: contactPage, PageURL: contactURL}, nil},
		{"a noindex page", audittest.Input{Draft: plain, PageHTML: audittest.Noindex(contactPage), PageURL: contactURL}, nil},
		// the kind comes from the address, so only the missing document keeps
		// this one quiet
		{"the page's text alone", audittest.Input{Draft: plain, PageText: "Contact us. Write to us any time.", PageURL: contactURL}, nil},
		{"no page", audittest.Input{Draft: plain, PageURL: contactURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, shop.PageKindType{}, tc.in, tc.want) })
	}
}

// TestPageTypeQuietAfterNamedType: a WebPage whose URL names the subtype its
// page's kind calls for is told once, by the name and URL check.
func TestPageTypeQuietAfterNamedType(t *testing.T) {
	in := audittest.Input{Draft: `{"@context":"https://schema.org","@type":"WebPage","url":"https://harbour.test/about/","name":"Harbour Jewellers"}`, PageHTML: aboutPage, PageURL: aboutURL}
	audittest.Expect(t, graph.NamedType{}, in, []audit.Issue{
		{Path: "$", Severity: audit.Warning, Message: `WebPage "Harbour Jewellers": its name or URL names the subtype AboutPage; type it AboutPage`},
	})
	audittest.Expect(t, shop.PageKindType{}, in, nil)
}

// TestPageTypeAfterNamedTwo: a name and URL that name two subtypes lead with
// only one of them, so the page's kind still says the other.
func TestPageTypeAfterNamedTwo(t *testing.T) {
	in := audittest.Input{Draft: `{"@context":"https://schema.org","@type":"WebPage","url":"https://harbour.test/contact/","name":"About and contact"}`, PageHTML: contactPage, PageURL: contactURL}
	audittest.Expect(t, graph.NamedType{}, in, []audit.Issue{
		{Path: "$", Severity: audit.Warning, Message: `WebPage "About and contact": its name or URL names the subtype AboutPage or ContactPage; type it AboutPage or ContactPage`},
	})
	audittest.Expect(t, shop.PageKindType{}, in, []audit.Issue{
		{Path: "$", Severity: audit.Warning, Message: "this WebPage is a contact page: type it ContactPage"},
	})
}
