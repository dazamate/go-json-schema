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
		{"a shipping time is no hours", audittest.Input{Draft: noDraft, PageHTML: hoursPage("Monday to Friday, orders ship in 2-3 days. Sale Monday 10-20% off. Buyers sat 5-6 to a table."), PageURL: contactURL}, nil},
		{"hours on a 24 hour clock", audittest.Input{Draft: noDraft, PageHTML: hoursPage("Mon-Fri 0900-1700"), PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives opening hours ("Mon-Fri 0900-1700"): ` + hoursAdvice},
		}},
		{"hours with a colon", audittest.Input{Draft: noDraft, PageHTML: hoursPage("Saturday 10:00 - 16:00"), PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives opening hours ("Saturday 10:00 - 16:00"): ` + hoursAdvice},
		}},
		{"hours with a bare first hour", audittest.Input{Draft: noDraft, PageHTML: hoursPage("Sun 10-4pm"), PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives opening hours ("Sun 10-4pm"): ` + hoursAdvice},
		}},
		{"closed on a day", audittest.Input{Draft: noDraft, PageHTML: hoursPage("Sunday: closed"), PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives opening hours ("Sunday: closed"): ` + hoursAdvice},
		}},
		{"hours in the site's footer only", audittest.Input{Draft: noDraft, PageHTML: `<!doctype html><html><head></head><body><main><h1>2015 Toyota Aqua</h1><p>Low kilometres, one owner.</p></main><footer><p>Showroom hours</p><p>Mon-Fri 8:30am-5:30pm</p></footer></body></html>`, PageURL: contactURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page gives opening hours ("Mon-Fri 8:30am-5:30pm"): ` + hoursAdvice},
		}},
		{"openingHours in the draft", audittest.Input{Draft: carried, PageHTML: contactPage, PageURL: contactURL}, nil},
		{"a page that gives no hours", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(contactPage), PageURL: contactURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: contactURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.OpeningHours{}, tc.in, tc.want) })
	}
}

// hoursAdvice is the end of the message every opening hours issue shares.
const hoursAdvice = `set the store's openingHoursSpecification, an OpeningHoursSpecification for each run of days (dayOfWeek, DayOfWeek members such as https://schema.org/Monday, opens, closes); an online shop's hours go on its ContactPoint as hoursAvailable`

// hoursPage is a contact page with the line in its main element.
func hoursPage(line string) string {
	return `<!doctype html><html><head></head><body><main><h1>Contact us</h1><p>` + line + `</p></main></body></html>`
}
