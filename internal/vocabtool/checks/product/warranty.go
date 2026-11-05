package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Warranty{}) }

var warranty = regexp.MustCompile(`(?i)\b(?:\d+|one|two|three|five|ten)[- ](?:years?|yrs?|months?)['’]?\s+warranty\b|\blifetime warranty\b|\bwarranty (?:of|for) \d+ (?:years?|months?)\b`)

// Warranty warns when a product page promises a warranty and the draft has no
// WarrantyPromise: the Offer carries the promise and its length.
type Warranty struct{}

// ID is the check's ID.
func (Warranty) ID() string { return "product/warranty" }

// Run reports the warranty the page promises.
func (Warranty) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || c.Has("WarrantyPromise") {
		return nil
	}
	m := warranty.FindString(c.Page.Main)
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page promises %q: set the Offer's warranty, a WarrantyPromise (durationOfWarranty a QuantitativeValue with unitCode ANN, warrantyScope)", m)}
}
