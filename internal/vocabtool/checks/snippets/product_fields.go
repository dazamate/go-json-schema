package snippets

import (
	"fmt"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(ProductFields{}) }

// ProductFields notes the offers, review and aggregateRating a Product lacks
// when it has one of them: Search Console only warns on these. Each takes a
// fact the page may not give, so the advice is to add offers only with the
// price the page shows, and a review or rating only from real ones, never
// invented: fake ratings break Google's review snippet guidelines and can
// bring a manual action. A Product with none of the three is
// ProductOffersOrReview's, and a ProductGroup is left out, as its variants
// carry them.
type ProductFields struct{}

// ID is the check's ID.
func (ProductFields) ID() string { return "snippets/product-fields" }

// Run reports each Product with some but not all of the three.
func (ProductFields) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, p := range productsOf(c) {
		missing := missingFields(p)
		if c.Is(p.N, "ProductGroup") || len(missing) == 0 || len(missing) == len(productFields) {
			continue
		}
		var fields []string
		for _, k := range missing {
			fields = append(fields, fmt.Sprintf(`Missing field %q`, k))
		}
		advice := ". Add offers only with the price the page shows"
		if has(p.N, "offers") {
			advice = `. Add review and aggregateRating only from real reviews and ratings the page shows, never invented ones: fake ratings break Google's review snippet guidelines and can bring a manual action`
		}
		out = append(out, audit.Issuef(p.Path, audit.Info, "%s%s%s", nonCritical, strings.Join(fields, "; "), advice))
	}
	return out
}
