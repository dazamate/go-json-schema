package site

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(OnlineBusiness{}) }

var _ audit.Retyper = OnlineBusiness{}

// OnlineBusiness warns on a site's publishing Organization, typed no more
// specifically, that shows no premises: no address, location or hours in the
// markup, and none anywhere on the page, its footer included. It works
// through its website, so it is an OnlineBusiness. A shop's business checks
// type a shop. It leaves be a node an earlier source in the retype ledger
// told. It needs the page's document, on a page search engines index.
type OnlineBusiness struct{}

// ID is the check's ID.
func (OnlineBusiness) ID() string { return "site/online-business" }

// premises are the properties that place a business.
var premises = []string{"address", "location", "geo", "openingHours", "openingHoursSpecification", "hasMap"}

// find is every publishing Organization with a URL that shows no premises,
// on a site that sells nothing and whose page shows no street address or
// opening hours, that no source before it told.
func (OnlineBusiness) find(c *audit.Context, before audit.Retyped) []jsonld.Node {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	if c.Page.Sells() || webpage.StreetAddress.MatchString(c.Page.Text) || webpage.OpeningTime.MatchString(c.Page.Text) {
		return nil
	}
	var out []jsonld.Node
	for _, l := range jsonld.Publishers(c.Nodes) {
		if c.Single(l.N) != "Organization" || before.Told[l.N] || len(jsonld.Strs(l.N, "url")) == 0 {
			continue
		}
		if slices.ContainsFunc(premises, func(k string) bool { _, ok := l.N.Get(k); return ok }) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// Run reports each such Organization.
func (k OnlineBusiness) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, l := range k.find(c, c.RetypedBefore(k.ID())) {
		name := txt.OrDefault(strings.Join(jsonld.Strs(l.N, "name"), " "), "the business")
		out = append(out, audit.Issuef(l.Path, audit.Warning, "Organization %q shows no premises and works through its website: type it OnlineBusiness", name))
	}
	return out
}

// Retypes tells each such Organization and names OnlineBusiness.
func (k OnlineBusiness) Retypes(c *audit.Context, before audit.Retyped) []audit.Retype {
	var out []audit.Retype
	for _, l := range k.find(c, before) {
		out = append(out, audit.Retype{Node: l.N, Named: []string{"OnlineBusiness"}})
	}
	return out
}
