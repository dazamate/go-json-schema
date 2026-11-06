package snippets

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(ProductOffersOrReview{}) }

// ProductOffersOrReview warns about a Product with none of offers, review and
// aggregateRating: Search Console marks the item invalid until it has one. A
// ProductGroup is left out, as its variants carry them. The advice is to show
// the price, never to invent a rating: with none of the three, the node may
// not be a product snippet at all.
type ProductOffersOrReview struct{}

// ID is the check's ID.
func (ProductOffersOrReview) ID() string { return "snippets/product-offers-or-review" }

// Run reports each Product with none of the three.
func (ProductOffersOrReview) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, p := range productsOf(c) {
		if c.Is(p.N, "ProductGroup") || len(missingFields(p)) != len(productFields) {
			continue
		}
		kind := "a Product"
		if p.ty != "Product" {
			kind = "a " + p.ty + ", a kind of Product, so the report reads it as one"
		}
		out = append(out, audit.Issuef(p.Path, audit.Warning, critical+`Either "offers", "review", or "aggregateRating" should be specified. This node is %s, and the item is invalid. Give it offers with the price the page shows for it (an Offer, or an AggregateOffer for a range), or review or aggregateRating only from real reviews and ratings the page shows. If the page shows none of them, it is not a product snippet: show the price, or type the node outside Product`, kind))
	}
	return out
}

// productFields are the fields a Product snippet takes beside its name.
var productFields = []string{"offers", "review", "aggregateRating"}

// missingFields are the product's fields of productFields that it lacks.
func missingFields(p product) []string {
	var out []string
	for _, k := range productFields {
		if !has(p.N, k) {
			out = append(out, k)
		}
	}
	return out
}
