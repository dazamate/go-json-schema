package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

// either is the finding's message for a node of the kind.
func either(kind string) string {
	return `Either "offers", "review", or "aggregateRating" should be specified. This node is ` + kind + `, and the item is invalid. Give it offers with the price the page shows for it (an Offer, or an AggregateOffer for a range), or review or aggregateRating only from real reviews and ratings the page shows. If the page shows none of them, it is not a product snippet: show the price, or type the node outside Product`
}

func TestProductOffersOrReview(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"a Product with nothing to show", `{"@type":"Product","name":"Small moving box"}`, []audit.Issue{
			warn("$", either("a Product")),
		}},
		// what Search Console flagged on liftmyboxes.com's truck picker
		{"a Vehicle is read as a Product", `{"@type":"ItemList","itemListElement":[{"@type":"Vehicle","name":"10 ft","cargoVolume":{"@type":"QuantitativeValue","minValue":402,"maxValue":450,"unitCode":"FTQ"}}]}`, []audit.Issue{
			warn("$.itemListElement[0]", either("a Vehicle, a kind of Product, so the report reads it as one")),
		}},
		{"offers", `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1}}`, nil},
		{"a review", `{"@type":"Product","name":"Box","review":` + review + `}`, nil},
		{"a rating", `{"@type":"Product","name":"Box","aggregateRating":` + rating + `}`, nil},
		// its variants carry the offers
		{"a ProductGroup", `{"@type":"ProductGroup","name":"Box","productGroupID":"box"}`, nil},
		{"a reference is checked where its node is", `{"@type":"WebPage","about":{"@type":"Product","@id":"https://example.com/#box"}}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.ProductOffersOrReview{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
