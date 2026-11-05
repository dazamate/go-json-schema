package policies_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

// An English-language business outside AU, NZ, US, GB and CA names its
// country too.
func TestBasedInMoreCountries(t *testing.T) {
	for place, code := range map[string]string{"Ireland": "IE", "India": "IN", "South Africa": "ZA", "Singapore": "SG", "Scotland": "GB", "England": "GB"} {
		page := strings.Replace(aboutPage, "based in Australia", "based in "+place, 1)
		audittest.Expect(t, policies.BasedIn{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: aboutURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page says the business is based in " + place + ": set its address, a PostalAddress with addressCountry " + code},
		})
	}
}
