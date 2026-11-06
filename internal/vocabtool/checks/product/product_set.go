package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(ProductSet{}) }

var setOf = regexp.MustCompile(`(?i)\b(\d+)\s*(?:pcs?|pieces?)\b|\bset of (\d+|two|three|four|five|six)\b|\b(\d+|two|three|four|five|six)[- ]piece set\b|\bpack of (\d+|two|three|four|five|six)\b`)

// ProductSet warns when a product page's name or title says it sells a set
// of pieces and the draft has no ProductCollection: a set is a collection
// whose pieces are listed.
type ProductSet struct{}

// ID is the check's ID.
func (ProductSet) ID() string { return "product/product-set" }

// Run reports the set the page's name and title announce.
func (ProductSet) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || c.Has("ProductCollection") {
		return nil
	}
	m := setOf.FindStringSubmatch(headOf(c))
	if m == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page sells %q, a set of %s: type its Product a ProductCollection and list the pieces in includesObject, a TypeAndQuantityNode for each (typeOfGood, amountOfThisGood)", txt.Clip(c.Page.H1(), 80), txt.FirstNonEmpty(m[1:]...))}
}
