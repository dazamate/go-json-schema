package figures

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Quantities{}) }

var quantity = regexp.MustCompile(`(?i)(?:^|[^$\d.,])(\d[\d,]*(?:\.\d+)?(?:\s*(?:–|-|to)\s*\d[\d,]*(?:\.\d+)?)?)\s*(hours?|hrs?|minutes?|mins?|days?|weeks?|months?|years?|miles?|mi|km|lbs?|pounds?|kg|cu\.? ?ft|cubic feet|sq\.? ?ft|square feet|ft|feet|°F|°C)\b`)

// Quantities warns on measured quantities a tool's output or a data element
// gives (4–6.5 hours, 4,500–6,500 lb) whose numbers the markup does not hold:
// each is a QuantitativeValue. A shop's sizes are the product checks'. It
// needs the page's document, on a page search engines index.
type Quantities struct{}

// ID is the check's ID.
func (Quantities) ID() string { return "figures/quantities" }

// Run lists the quantities the markup lacks, in one issue.
func (Quantities) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || c.Page.Sells() {
		return nil
	}
	held := map[float64]bool{}
	for _, l := range c.Nodes {
		for _, k := range l.N.Keys {
			for _, s := range jsonld.Scalars(l.N, k) {
				for _, n := range txt.NumberIn.FindAllString(strings.ReplaceAll(s, ",", ""), -1) {
					if f, ok := txt.Amount(n); ok {
						held[f] = true
					}
				}
			}
		}
	}
	var list []string
	for _, n := range webpage.QueryAll(c.Page.Doc, "output, data") {
		if webpage.Chrome(n) {
			continue
		}
		for _, m := range quantity.FindAllStringSubmatch(webpage.HTMLText(n), -1) {
			q := strings.TrimSpace(m[1] + " " + m[2])
			all := true
			for _, s := range txt.NumberIn.FindAllString(strings.ReplaceAll(m[1], ",", ""), -1) {
				if f, _ := txt.Amount(s); !held[f] {
					all = false
				}
			}
			if !all && !slices.Contains(list, q) {
				list = append(list, q)
			}
		}
	}
	if len(list) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page's tools and figures give quantities (%s): mark each up as a QuantitativeValue (value, or minValue and maxValue, unitCode such as HUR, DAY, MON, SMI, LBR or FTQ) on the thing it measures; a property whose range is a Duration, Distance or Mass takes it as that type's text (PT4H, \"10 mi\", \"6500 lb\")", txt.Examples(list))}
}
