package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestAddress(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"LocalBusiness","address":{"@type":"PostalAddress","streetAddress":"12 Wharf Street"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a street address on the page", audittest.Input{Draft: noDraft, PageHTML: contactPage, PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page shows the address "12 Wharf Street, Fremantle": set the business's address, a PostalAddress (streetAddress, addressLocality, addressRegion, postalCode, addressCountry)`},
		}},
		{"an address in the draft", audittest.Input{Draft: carried, PageHTML: contactPage, PageURL: contactURL}, nil},
		{"a page that shows no address", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(contactPage), PageURL: contactURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: contactURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.Address{}, tc.in, tc.want) })
	}
}
