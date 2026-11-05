package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

var (
	// unitPrice is a price per a measure: in the currency's sign or in pence
	// ("12p per 100g"), per a weight, a volume or a length.
	unitPrice = regexp.MustCompile(`(?i)(?:(?:A\$|\$|£|€|₹)\s?\d+(?:\.\d{1,2})?|\b\d+(?:\.\d+)?p)\s*(?:per|/)\s*(?:\d+\s*)?(?:mg|g|kg|ml|cl|l|litres?|liters?|fl\.? ?oz|oz|ounces?|lbs?|pounds?|m|metres?|meters?)\b`)
	// addOn is a paid extra the shopper adds to the order: a gift wrap, an
	// engraving, a monogram, a gift box or bag, a warranty or protection plan.
	addOn = regexp.MustCompile(`(?i)\badd (?:an? |your |free )?(?:custom(?:ised|ized)? |personali[sz]ed |monogram(?:med)? )?(?:(?:\d+[- ](?:year|yr|month)s?[- ])?(?:extended |extra )?(?:warranty|protection plan|care plan|insurance)|gift[- ]?wrap(?:ping)?|engraving|gift box|gift bag|monogram(?:ming)?|personali[sz]ation)\b[^.$]{0,20}(?:A\$|\$|£|€)\s?\d+(?:\.\d{1,2})?`)
)

// extrasKey is the memo key of the page's extras.
type extrasKey struct{}

// extrasFound is the memoized extras: the unit price and the add-on.
type extrasFound struct{ unit, extra string }

// extras are the priced extras a page search engines index of a shop shows
// and no node carries, found once for the checks that share them: a unit
// price, unless the draft has a referenceQuantity, and a paid add-on, unless
// it has an addOn. Each is "" when there is none.
func extras(c *audit.Context) (unit, extra string) {
	f := audit.Memo(c, extrasKey{}, func() extrasFound {
		if !c.Indexed() || !c.Page.Sells() {
			return extrasFound{}
		}
		var f extrasFound
		if m := unitPrice.FindString(c.Page.Main); m != "" && !jsonld.HasKey(c.Nodes, "referenceQuantity") {
			f.unit = m
		}
		if m := addOn.FindString(c.Page.Main); m != "" && !jsonld.HasKey(c.Nodes, "addOn") {
			f.extra = m
		}
		return f
	})
	return f.unit, f.extra
}
