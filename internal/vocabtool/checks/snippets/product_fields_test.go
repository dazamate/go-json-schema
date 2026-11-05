package snippets_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/snippets"
)

func TestProductFields(t *testing.T) {
	const offers = `{"@type":"Offer","price":1,"priceCurrency":"USD","availability":"InStock"}`
	const addOffers = `. Add offers only with the price the page shows`
	const addReviews = `. Add review and aggregateRating only from real reviews and ratings the page shows, never invented ones: fake ratings break Google's review snippet guidelines and can bring a manual action`
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"offers alone", `{"@type":"Product","name":"Box","offers":` + offers + `}`, []audit.Issue{
			info("$", `Missing field "review"; Missing field "aggregateRating"`+addReviews),
		}},
		{"offers and a review", `{"@type":"Product","name":"Box","offers":` + offers + `,"review":` + review + `}`, []audit.Issue{
			info("$", `Missing field "aggregateRating"`+addReviews),
		}},
		{"ratings without offers", `{"@type":"Product","name":"Box","aggregateRating":` + rating + `}`, []audit.Issue{
			info("$", `Missing field "offers"; Missing field "review"`+addOffers),
		}},
		{"a Vehicle is read as a Product", `{"@type":"Vehicle","name":"10 ft","review":` + review + `}`, []audit.Issue{
			info("$", `Missing field "offers"; Missing field "aggregateRating"`+addOffers),
		}},
		{"everything the snippet takes", `{"@type":"Product","name":"Box","offers":` + offers + `,"review":` + review + `,"aggregateRating":` + rating + `}`, nil},
		// the critical check asks for one of them
		{"none of them", `{"@type":"Product","name":"Box"}`, nil},
		// its variants carry the offers
		{"a ProductGroup", `{"@type":"ProductGroup","name":"Box","productGroupID":"box","review":` + review + `}`, nil},
		{"a reference is checked where its node is", `{"@type":"WebPage","about":{"@type":"Product","@id":"https://example.com/#box"}}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, snippets.ProductFields{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
