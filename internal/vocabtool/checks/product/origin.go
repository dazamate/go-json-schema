package product

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Origin{}) }

// place is a place name, or a place in a country ("County Dublin, Ireland").
const place = `[A-Z][a-z]+(?: [A-Z][a-z]+)?(?:, [A-Z][a-z]+(?: [A-Z][a-z]+)?)?`

var madeIn = regexp.MustCompile(`\b(?:[Hh]and(?:made|crafted)|[Mm]ade|[Cc]rafted|[Mm]anufactured) in (?:the )?(` + place + `)|\b[Cc]ountry of [Oo]rigin:?\s+([A-Z][a-z]+(?: (?:States|Kingdom|Kong|Zealand|Africa|Korea|Arabia|Emirates|Republic|Lanka|Rico))?)`)

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
	where := txt.FirstNonEmpty(m[1:]...)
	if _, country, ok := strings.Cut(where, ", "); ok {
		where = country
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set the Product's countryOfOrigin, a Country (name %s)", m[0], where)}
}
