package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

var (
	unitPrice = regexp.MustCompile(`(?i)(?:A\$|\$|£|€)\s?\d+(?:\.\d{1,2})?\s*(?:per|/)\s*(?:\d+\s*)?(?:g|kg|ml|l|litres?|liters?|m|metres?|meters?)\b`)
	addOn     = regexp.MustCompile(`(?i)\badd (?:gift[- ]?wrap(?:ping)?|engraving|a gift box|gift box|personali[sz]ation)\b[^.$]{0,20}(?:A\$|\$|£|€)\s?\d+(?:\.\d{1,2})?`)
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
