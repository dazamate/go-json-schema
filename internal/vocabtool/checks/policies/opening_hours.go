package policies

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"regexp"
)

func init() { audit.Register(OpeningHours{}) }

// alwaysOpen finds a statement of when a business is open that names no day
// or hour: "Open 24 hours a day, 7 days a week", "open 7 days", "open daily".
var alwaysOpen = regexp.MustCompile(`(?i)\bopen (?:24 hours(?: a day)?(?:,? (?:and )?7 days(?: a week)?)?|24/7|7 days(?: a week)?|daily|every day|all day)\b`)

// OpeningHours warns when a page gives the business's opening hours and no
// node has openingHoursSpecification, openingHours or hoursAvailable.
type OpeningHours struct{}

// ID is the check's ID.
func (OpeningHours) ID() string { return "policies/opening-hours" }

// Run reports the first opening time the page gives.
func (OpeningHours) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	m := c.Page.Hours()
	if m == "" {
		// a business that never closes says so in its header or footer
		m = alwaysOpen.FindString(c.Page.Text)
	}
	if m == "" || jsonld.HasKey(c.Nodes, "openingHoursSpecification") || jsonld.HasKey(c.Nodes, "openingHours") || jsonld.HasKey(c.Nodes, "hoursAvailable") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page gives opening hours (%q): set the store's openingHoursSpecification, an OpeningHoursSpecification for each run of days (dayOfWeek, DayOfWeek members such as https://schema.org/Monday, opens, closes); an online shop's hours go on its ContactPoint as hoursAvailable", m)}
}
