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

// extras are the priced extras a page search engines index of a shop shows
// and no node carries: a unit price, unless the draft has a
// referenceQuantity, and a paid add-on, unless it has an addOn. Each is ""
// when there is none.
func extras(c *audit.Context) (unit, extra string) {
	if !c.Indexed() || !c.Page.Sells() {
		return "", ""
	}
	if m := unitPrice.FindString(c.Page.Main); m != "" && !jsonld.HasKey(c.Nodes, "referenceQuantity") {
		unit = m
	}
	if m := addOn.FindString(c.Page.Main); m != "" && !jsonld.HasKey(c.Nodes, "addOn") {
		extra = m
	}
	return unit, extra
}
