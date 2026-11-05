package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestContactPoint(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"Organization","contactPoint":{"@type":"ContactPoint","contactType":"customer service","email":"hello@harbour.test"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a contact page", audittest.Input{Draft: noDraft, PageHTML: contactPage, PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page is the business's contact page: give the business a contactPoint, a ContactPoint (contactType customer service, email, telephone, areaServed, availableLanguage)"},
		}},
		{"a ContactPoint in the draft", audittest.Input{Draft: carried, PageHTML: contactPage, PageURL: contactURL}, nil},
		{"a page that is no contact page", audittest.Input{Draft: noDraft, PageHTML: asVisit(contactPage), PageURL: "https://harbour.test/pages/visit"}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(contactPage), PageURL: contactURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: contactURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.ContactPoint{}, tc.in, tc.want) })
	}
}
