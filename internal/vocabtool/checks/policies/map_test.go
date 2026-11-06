package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestMap(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"LocalBusiness","geo":{"@type":"GeoCoordinates","latitude":-32.05,"longitude":115.74}}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"an embedded map", audittest.Input{Draft: noDraft, PageHTML: contactPage, PageURL: contactURL}, []audit.Issue{
			{Path: "html:body > main > iframe", Severity: audit.Warning, Message: "the page embeds a map of the business: set its hasMap (the map's URL) and geo, GeoCoordinates (latitude, longitude)"},
		}},
		{"geo in the draft", audittest.Input{Draft: carried, PageHTML: contactPage, PageURL: contactURL}, nil},
		{"a page with no map", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: "Visit the workshop at 12 Wharf Street, Fremantle.", PageURL: contactURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(contactPage), PageURL: contactURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: contactURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.Map{}, tc.in, tc.want) })
	}
}
