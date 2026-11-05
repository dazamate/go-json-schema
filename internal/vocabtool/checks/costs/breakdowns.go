package costs

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Breakdowns{}) }

var goodsNoun = regexp.MustCompile(`(?i)\b(?:boxes|box|tape|packing paper|bubble wrap|mattress bags?|moving blankets?|furniture pads?|dollies|dolly|stretch wrap|packing peanuts)\b`)

// Breakdowns warns on a total the page breaks into priced parts: a list,
// definition list or table of two or more priced lines beside the total (an
// output or data element). The total is a CompoundPriceSpecification and
// each part its priceComponent.
type Breakdowns struct{}

// ID is the check's ID.
func (Breakdowns) ID() string { return "costs/breakdowns" }

// breakdown is a total the page breaks into priced parts.
type breakdown struct {
	box   *html.Node // the block that holds the total and its parts
	label string     // the block's heading, else "the total"
	total string     // the total's text
	parts []string   // the priced parts' text
}

// find is the page's breakdowns whose amounts are fresh, given those taken
// before; each takes its amounts as it is found, so a breakdown the page
// repeats is found once. took is every amount they take.
func (Breakdowns) find(c *audit.Context, before audit.Taken) (found []breakdown, took audit.Taken) {
	if !costPage(c) || c.Page.Doc == nil {
		return nil, nil
	}
	took = audit.Taken{}
	for _, list := range webpage.QueryAll(c.Page.Doc, "dl, ul, ol, table") {
		if webpage.Chrome(list) || !webpage.Shown(list) {
			continue
		}
		var parts []string
		switch list.DataAtom {
		case atom.Dl:
			for _, d := range webpage.Pairs(list) {
				if txt.PriceAmount.MatchString(d.Value) {
					parts = append(parts, d.Value)
				}
			}
		case atom.Table:
			for _, tr := range webpage.QueryAll(list, "tr") {
				if t := webpage.HTMLText(tr); txt.PriceAmount.MatchString(t) && len(webpage.QueryAll(tr, "td")) > 0 {
					parts = append(parts, t)
				}
			}
		default:
			for _, li := range webpage.QueryAll(list, "li") {
				if t := webpage.HTMLText(li); txt.PriceAmount.MatchString(t) {
					parts = append(parts, t)
				}
			}
		}
		// a list of goods' prices is the goods check's
		goods := 0
		for _, t := range parts {
			if goodsNoun.MatchString(t) {
				goods++
			}
		}
		if len(parts) < 2 || goods*2 >= len(parts) {
			continue
		}
		// the total sits in the list's own block, at most its article or
		// section
		var box, total *html.Node
		for e, up := list.Parent, 0; e != nil && up < 3 && total == nil && e.DataAtom != atom.Main && e.DataAtom != atom.Body; e, up = e.Parent, up+1 {
			for _, o := range webpage.QueryAll(e, "output, data[value]") {
				if !slices.Contains(webpage.Ancestors(o), list) && txt.PriceAmount.MatchString(webpage.HTMLText(o)) {
					box, total = e, o
					break
				}
			}
			if e.DataAtom == atom.Article || e.DataAtom == atom.Section {
				break
			}
		}
		if total == nil {
			continue
		}
		shown := webpage.HTMLText(total) + " " + strings.Join(parts, " ")
		if !fresh(c, shown, before, took) {
			continue
		}
		label := "the total"
		for e := box; e != nil && e.DataAtom != atom.Body; e = e.Parent {
			if hs := webpage.QueryAll(e, "h2, h3, h4, h5"); len(hs) > 0 {
				label = webpage.HTMLText(hs[0])
				break
			}
		}
		took.Take(shown)
		found = append(found, breakdown{box: box, label: label, total: webpage.HTMLText(total), parts: parts})
	}
	return found, took
}

// Run reports each breakdown at its block.
func (b Breakdowns) Run(c *audit.Context) []audit.Issue {
	found, _ := b.find(c, c.TakenBefore(b.ID()))
	var out []audit.Issue
	for _, f := range found {
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, f.box), audit.Warning, "the page breaks %q (%s) into %d priced parts: mark it up as a CompoundPriceSpecification (the PriceSpecification that sums its parts: minPrice, maxPrice, priceCurrency) with each part a priceComponent (a UnitPriceSpecification with priceComponentType such as DistanceFee for mileage, or a DeliveryChargeSpecification for a delivery or pickup charge)", f.label, f.total, len(f.parts)))
	}
	return out
}

// TakesPrices takes the amounts of the totals and parts it reports.
func (b Breakdowns) TakesPrices(c *audit.Context, before audit.Taken) audit.Taken {
	_, took := b.find(c, before)
	return took
}
