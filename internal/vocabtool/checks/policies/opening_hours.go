package policies

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(OpeningHours{}) }

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
	m := webpage.OpeningTime.FindString(c.Page.Main)
	if m == "" || jsonld.HasKey(c.Nodes, "openingHoursSpecification") || jsonld.HasKey(c.Nodes, "openingHours") || jsonld.HasKey(c.Nodes, "hoursAvailable") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page gives opening hours (%q): set the store's openingHoursSpecification, an OpeningHoursSpecification for each run of days (dayOfWeek, DayOfWeek members such as https://schema.org/Monday, opens, closes); an online shop's hours go on its ContactPoint as hoursAvailable", m)}
}
