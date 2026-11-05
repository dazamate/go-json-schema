package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(Origin{}) }

var madeIn = regexp.MustCompile(`\b(?:[Hh]and(?:made|crafted)|[Mm]ade|[Cc]rafted|[Mm]anufactured) in (?:the )?([A-Z][a-z]+(?: [A-Z][a-z]+)?)`)

// Origin warns when a product page says where the product is made and no
// node has countryOfOrigin.
type Origin struct{}

// ID is the check's ID.
func (Origin) ID() string { return "product/origin" }

// Run reports the place the page says the product is made.
func (Origin) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || jsonld.HasKey(c.Nodes, "countryOfOrigin") {
		return nil
	}
	m := madeIn.FindStringSubmatch(c.Page.Main)
	if m == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set the Product's countryOfOrigin, a Country (name %s)", m[0], m[1])}
}
