package product

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Sizes{}) }

var sizeMeasure = regexp.MustCompile(`\b(\d{1,2}(?:\.\d)?|XXS|XS|XXL|XL|[SML])\s*[-–:=]\s*\(?\s*(\d+(?:\.\d+)?)\s*(mm|cm)\b`)

// Sizes warns when a product page gives its sizes with their measurements (6
// - 16mm inner diameter) and no SizeSpecification carries them.
type Sizes struct{}

// ID is the check's ID.
func (Sizes) ID() string { return "product/sizes" }

// Run reports the sizes the page measures, when it gives at least two.
func (Sizes) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) {
		return nil
	}
	ms := sizeMeasure.FindAllStringSubmatch(c.Page.Main, -1)
	if len(ms) < 2 || c.Has("SizeSpecification") {
		return nil
	}
	var list []string
	for _, m := range ms {
		if s := m[1] + ": " + m[2] + m[3]; !slices.Contains(list, s) {
			list = append(list, s)
		}
	}
	if len(list) > 4 {
		list = append(list[:4:4], "…")
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page gives sizes with their measurements (%s): make each variant's size a SizeSpecification (name, hasMeasurement a QuantitativeValue with value and unitCode, sizeSystem a WearableSizeSystemEnumeration member such as WearableSizeSystemUS, sizeGroup a WearableSizeGroupEnumeration member)", strings.Join(list, ", "))}
}
