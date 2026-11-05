package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(OneOfAKind{}) }

var oneOfAKind = regexp.MustCompile(`(?i)\bone[- ]of[- ]a[- ]kind\b|\bOOAK\b|\bunique piece\b|\bonly one (?:made|available)\b`)

// OneOfAKind warns when a product page says its piece is one of a kind and
// the draft has no IndividualProduct: a single piece is an IndividualProduct,
// with its serial number if it has one.
type OneOfAKind struct{}

// ID is the check's ID.
func (OneOfAKind) ID() string { return "product/one-of-a-kind" }

// Run reports the words that make the piece one of a kind.
func (OneOfAKind) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || c.Has("IndividualProduct") {
		return nil
	}
	m := oneOfAKind.FindString(headOf(c) + " · " + c.Page.Main)
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: type the Product IndividualProduct (with its serialNumber if it has one)", m)}
}
