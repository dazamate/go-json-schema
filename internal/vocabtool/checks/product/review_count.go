package product

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(ReviewCount{}) }

var reviewCount = regexp.MustCompile(`(?i)\b(\d[\d,]*)\s+(?:customer\s+|product\s+)?reviews?\b`)

// ReviewCount warns when a page counts its reviews ("128 reviews") and no
// node has an aggregateRating with that count: the count is the rating's
// reviewCount.
type ReviewCount struct{}

// ID is the check's ID.
func (ReviewCount) ID() string { return "product/review-count" }

// Run reports the first review count the page shows that no reviewCount or
// ratingCount holds.
func (ReviewCount) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	m := reviewCount.FindStringSubmatch(c.Page.Main)
	if m == nil {
		return nil
	}
	n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
	counts := append(jsonld.Under(c.Nodes, "reviewCount"), jsonld.Under(c.Nodes, "ratingCount")...)
	if n <= 0 || txt.HoldsNumber(counts, float64(n)) {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page counts %d reviews: set the %s's aggregateRating, an AggregateRating (ratingValue, bestRating, reviewCount %d)", n, reviewed(c), n)}
}

// ratedTypes are the types a rating is for, the page's own subject first and
// the business that publishes it last.
var ratedTypes = []string{"Recipe", "Product", "Course", "Book", "Movie", "SoftwareApplication", "Event", "Service", "LocalBusiness", "Organization"}

// reviewed is the type of the node the reviews are of: the draft's node of the
// first rated type, by its own type when it has one; a Product when the draft
// has none.
func reviewed(c *audit.Context) string {
	for _, ty := range ratedTypes {
		for _, l := range c.Nodes {
			if c.Is(l.N, ty) {
				return txt.OrDefault(c.Single(l.N), ty)
			}
		}
	}
	return "Product"
}
