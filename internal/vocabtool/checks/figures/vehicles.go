package figures

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Vehicles{}) }

var (
	vehicleKind = regexp.MustCompile(`(?i)\b(?:trucks|vans|vehicles)\b`)
	vehicleName = regexp.MustCompile(`(?i)\b(?:van|truck|trailer|pickup|lorry|ute|box truck|cube)\b`)
	capacity    = regexp.MustCompile(`(?i)\bcu\.? ?ft\b|\bcubic\b|\bpayload\b|\brated load\b|\bcapacity\b|\bholds?\b|\bspace\b|\blb\b|\bkg\b`)
)

// Vehicles warns on a table that lists vehicles with their capacity (truck
// sizes and what each holds) when no node names them. A table whose caption
// or header names vehicles lists one in each row (10 ft, 15 ft). Its ask
// carries the rule for a Vehicle, a kind of Product: it needs offers. It
// needs the page's document, on a page search engines index.
type Vehicles struct{}

// ID is the check's ID.
func (Vehicles) ID() string { return "figures/vehicles" }

// Run reports the page's first such table.
func (Vehicles) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p, d := c.Page, c.Index()
	for _, tbl := range webpage.QueryAll(p.Doc, "table") {
		if webpage.Chrome(tbl) || !capacity.MatchString(webpage.HTMLText(tbl)) {
			continue
		}
		head := ""
		for _, n := range webpage.QueryAll(tbl, "caption, thead, tr:first-child th") {
			head += " " + webpage.HTMLText(n)
		}
		every := vehicleName.MatchString(head) || vehicleKind.MatchString(head)
		var names []string
		for _, tr := range webpage.QueryAll(tbl, "tr") {
			cells := webpage.QueryAll(tr, "th, td")
			if len(cells) < 2 || len(webpage.QueryAll(tr, "td")) == 0 {
				continue
			}
			if n := webpage.HTMLText(cells[0]); (every || vehicleName.MatchString(n)) && len(strings.Fields(n)) <= 5 && !d.HoldsText(n) {
				names = append(names, n)
			}
		}
		if len(names) < 2 {
			continue
		}
		name := ""
		if cs := webpage.QueryAll(tbl, "caption"); len(cs) > 0 {
			name = webpage.HTMLText(cs[0])
		}
		at := tbl
		if b := webpage.IDBox(tbl); b != nil && b.DataAtom != 0 && slices.Contains([]string{"section", "article", "div", "figure"}, b.Data) {
			at = b
		}
		return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, at), audit.Warning, "the page's table %q lists vehicles (%s) with their capacity: add a Vehicle for each (name, cargoVolume, payload, vehicleConfiguration), in an ItemList or as the items of the Offer or Service they are for. A Vehicle is a kind of Product, and Search Console's Product snippets report marks one without offers, review or aggregateRating invalid: give each offers with the rental or sale price the page shows for it (an AggregateOffer for a range: lowPrice, highPrice, priceCurrency, offerCount)", name, txt.Examples(names))}
	}
	return nil
}
