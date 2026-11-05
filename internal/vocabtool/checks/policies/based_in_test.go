package policies_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestBasedIn(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"LocalBusiness","address":{"@type":"PostalAddress","addressCountry":"AU"}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"an about page says where the business is based", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page says the business is based in Australia: set its address, a PostalAddress with addressCountry AU"},
		}},
		{"an address in the draft", audittest.Input{Draft: carried, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a street address on the page", audittest.Input{Draft: noDraft, PageHTML: withStreet(aboutPage), PageURL: aboutURL}, nil},
		{"a page that is no contact or about page", audittest.Input{Draft: noDraft, PageHTML: asNotes(aboutPage), PageURL: "https://harbour.test/pages/studio"}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(aboutPage), PageURL: aboutURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: aboutURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.BasedIn{}, tc.in, tc.want) })
	}
}

// withStreet adds a street address to the page.
func withStreet(page string) string {
	return strings.Replace(page, "</main>", "<p>Find us at 12 Wharf Street.</p></main>", 1)
}
