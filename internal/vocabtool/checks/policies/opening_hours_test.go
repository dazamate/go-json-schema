package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestOpeningHours(t *testing.T) {
	carried := `{"@context":"https://schema.org","@type":"Store","openingHours":"Mo-Fr 09:00-17:00"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"hours on the page", audittest.Input{Draft: noDraft, PageHTML: contactPage, PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives opening hours ("Monday to Friday 9am to 5pm"): set the store's openingHoursSpecification, an OpeningHoursSpecification for each run of days (dayOfWeek, DayOfWeek members such as https://schema.org/Monday, opens, closes); an online shop's hours go on its ContactPoint as hoursAvailable`},
		}},
		{"openingHours in the draft", audittest.Input{Draft: carried, PageHTML: contactPage, PageURL: contactURL}, nil},
		{"a page that gives no hours", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(contactPage), PageURL: contactURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: contactURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.OpeningHours{}, tc.in, tc.want) })
	}
}
