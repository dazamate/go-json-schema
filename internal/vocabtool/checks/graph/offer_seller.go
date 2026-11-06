package graph

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(OfferSeller{}) }

// OfferSeller warns about an Offer that names no seller. An AggregateOffer
// gathers many sellers' offers, so it needs none. It needs the page's facts
// but not an indexed page: the markup's own faults are held to a page search
// engines skip too.
type OfferSeller struct{}

// ID is the check's ID.
func (OfferSeller) ID() string { return "graph/offer-seller" }

// Run reports each Offer with neither seller nor offeredBy.
func (OfferSeller) Run(c *audit.Context) []audit.Issue {
	if c.Page == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		if !c.Is(l.N, "Offer") || c.Is(l.N, "AggregateOffer") {
			continue
		}
		_, s := l.N.Get("seller")
		_, o := l.N.Get("offeredBy")
		if !s && !o {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "this Offer names no seller: set seller (or offeredBy) to the business that makes it"))
		}
	}
	return out
}
