package product

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Details{}) }

// Details warns on product details a product page lists (material, size,
// weight, a code) that the Product does not carry, in the property the
// label names or as an additionalProperty PropertyValue of that name.
type Details struct{}

// ID is the check's ID.
func (Details) ID() string { return "product/details" }

// Run reports every detail the draft lacks, once, in one message. A detail
// with no letter or digit in its value ("-") is no detail.
func (Details) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) {
		return nil
	}
	prods := productsOf(c)
	var missing []string
	for _, d := range productDetails(c.Page) {
		if !strings.ContainsFunc(d.Value, unicode.IsLetter) && !strings.ContainsFunc(d.Value, unicode.IsDigit) {
			continue
		}
		prop := detailProp(d.Label)
		if carriesDetail(c, prods, prop, d) {
			continue
		}
		where := prop
		if prop == "additionalProperty" {
			where = "additionalProperty, a PropertyValue with name, value and unitCode"
		}
		missing = append(missing, fmt.Sprintf("%s %q (%s)", d.Label, txt.Clip(d.Value, 40), where))
	}
	if len(missing) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page lists product details the markup lacks: %s; set each on the Product", strings.Join(missing, ", "))}
}
