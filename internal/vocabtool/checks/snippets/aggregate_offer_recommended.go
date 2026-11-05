package snippets

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(AggregateOfferRecommended{}) }

// AggregateOfferRecommended notes an AggregateOffer of a Product that lacks
// highPrice or offerCount: Search Console only warns on these. offerCount is
// how many sellers or sources the range comes from, a fact the page may not
// give.
type AggregateOfferRecommended struct{}

// ID is the check's ID.
func (AggregateOfferRecommended) ID() string { return "snippets/aggregate-offer-recommended" }

// Run reports each recommended field a Product's AggregateOffer lacks.
func (AggregateOfferRecommended) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, o := range offersOf(c) {
		if !c.Is(o.n, "AggregateOffer") {
			continue
		}
		for _, k := range []string{"highPrice", "offerCount"} {
			if !has(o.n, k) {
				what := "the highest price the range reaches"
				if k == "offerCount" {
					what = "how many offers the range spans: the sellers or sources it comes from"
				}
				out = append(out, audit.Issuef(o.path, audit.Info, nonCritical+`Missing field %q (in "offers"): give the AggregateOffer %s, %s`, k, k, what))
			}
		}
	}
	return out
}
