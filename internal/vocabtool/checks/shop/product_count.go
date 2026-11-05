package shop

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(ProductCount{}) }

var productCount = regexp.MustCompile(`(?i)\b(\d{1,3}(?:,\d{3})*|\d+)\s+(?:products|items|pieces|results|styles|designs)\b`)

// ProductCount warns on a listing's product count ("1,153 pieces") that no
// ItemList has as numberOfItems. A count below the cards the page shows is
// some other count.
type ProductCount struct{}

// ID is the check's ID.
func (ProductCount) ID() string { return "shop/product-count" }

// Run reports the listing's first product count when no node has it.
func (ProductCount) Run(c *audit.Context) []audit.Issue {
	cards := listed(c)
	if cards == nil || c.Page.Kind != webpage.KindListing {
		return nil
	}
	m := productCount.FindStringSubmatch(c.Page.Main)
	if m == nil {
		return nil
	}
	n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
	if n < len(cards) || slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool {
		return slices.Contains(jsonld.Scalars(l.N, "numberOfItems"), strconv.Itoa(n))
	}) {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set numberOfItems %d on the ItemList of its products", m[0], n)}
}
