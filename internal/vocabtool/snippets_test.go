package vocabtool

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// snippetIssues are the product snippet findings for a draft, by path.
func snippetIssues(t *testing.T, draft string) []Issue {
	t.Helper()
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var out []Issue
	for _, i := range r.Issues {
		if strings.HasPrefix(i.Message, "Search Console (Product snippets)") {
			// critical findings are warnings; non-critical ones, which may take facts the page lacks, info
			want := Warning
			if strings.HasPrefix(i.Message, "Search Console (Product snippets), non-critical") {
				want = Info
			}
			if i.Severity != want {
				t.Errorf("%s: severity %s, want %s: %s", i.Path, i.Severity, want, i.Message)
			}
			out = append(out, i)
		}
	}
	return out
}

func TestProductSnippets(t *testing.T) {
	for _, tc := range []struct {
		name, draft string
		want        []string // path + " " + message pattern, each matching one issue
	}{
		{
			name:  "a Product with nothing to show",
			draft: `{"@type":"Product","name":"Small moving box"}`,
			want:  []string{`^\$ critical: Either "offers", "review", or "aggregateRating" should be specified\. This node is a Product,`},
		},
		{
			// what Search Console flagged on liftmyboxes.com's truck picker
			name:  "a Vehicle is read as a Product",
			draft: `{"@type":"ItemList","itemListElement":[{"@type":"Vehicle","name":"10 ft","cargoVolume":{"@type":"QuantitativeValue","minValue":402,"maxValue":450,"unitCode":"FTQ"}}]}`,
			want:  []string{`^\$\.itemListElement\[0\] critical: Either "offers", "review", or "aggregateRating" should be specified\. This node is a Vehicle, a kind of Product`},
		},
		{
			// and on its box estimator
			name:  "a price range without offerCount, ratings or reviews",
			draft: `{"@type":"Product","name":"Small moving box","offers":{"@type":"AggregateOffer","lowPrice":1.12,"highPrice":2.08,"priceCurrency":"USD"}}`,
			want: []string{
				`^\$ non-critical: Missing field "review"; Missing field "aggregateRating"\. Add review and aggregateRating only from real reviews`,
				`^\$\.offers non-critical: Missing field "offerCount" \(in "offers"\)`,
			},
		},
		{
			name:  "an AggregateOffer without its required range",
			draft: `{"@type":"Product","name":"Box","review":{"@type":"Review","author":{"@type":"Person","name":"Ann Lee"},"reviewBody":"Sturdy."},"aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3},"offers":{"@type":"AggregateOffer","highPrice":2,"offerCount":3}}`,
			want: []string{
				`^\$\.offers critical: Missing field "lowPrice" \(in "offers"\)`,
				`^\$\.offers critical: Missing field "priceCurrency" \(in "offers"\)`,
			},
		},
		{
			name:  "an Offer with no price anywhere",
			draft: `{"@type":"Product","name":"Box","offers":[{"@type":"Offer","priceCurrency":"USD","availability":"InStock","priceSpecification":{"@type":"UnitPriceSpecification","minPrice":1,"maxPrice":2}}]}`,
			want: []string{
				`^\$ non-critical: Missing field "review"; Missing field "aggregateRating"`,
				`^\$\.offers\[0\] critical: Missing field "price" \(in "offers"\)`,
				`^\$\.offers\[0\]\.priceSpecification critical: Missing field "price" \(in "offers\.priceSpecification"\)`,
			},
		},
		{
			name:  "an Offer priced through its priceSpecification",
			draft: `{"@type":"Product","name":"Box","offers":{"@type":"Offer","availability":"InStock","priceSpecification":{"@type":"UnitPriceSpecification","price":1.5,"priceCurrency":"USD"}}}`,
			want:  []string{`^\$ non-critical: Missing field "review"; Missing field "aggregateRating"`},
		},
		{
			name: "offers given by reference",
			draft: `[{"@type":"Product","name":"Box","offers":{"@id":"#range"}},
				{"@type":"AggregateOffer","@id":"#range","lowPrice":1,"highPrice":2,"priceCurrency":"USD"}]`,
			want: []string{
				`^\$\.@graph\[0\] non-critical: Missing field "review"; Missing field "aggregateRating"`,
				`^\$\.@graph\[0\]\.offers non-critical: Missing field "offerCount" \(in "offers"\)`,
			},
		},
		{
			name:  "ratings without offers",
			draft: `{"@type":"Product","name":"Box","aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}}`,
			want:  []string{`^\$ non-critical: Missing field "offers"; Missing field "review"\. Add offers only with the price the page shows$`},
		},
		{
			name:  "a Product with no name",
			draft: `{"@type":"Product","offers":{"@type":"Offer","price":1,"priceCurrency":"USD","availability":"InStock"},"review":{"@type":"Review","author":{"@type":"Person","name":"Ann Lee"}},"aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}}`,
			want:  []string{`^\$ critical: Missing field "name"`},
		},
		{
			name:  "everything the snippet takes",
			draft: `{"@type":"Product","name":"Box","offers":{"@type":"Offer","price":1,"priceCurrency":"USD","availability":"InStock"},"review":{"@type":"Review","author":{"@type":"Person","name":"Ann Lee"}},"aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}}`,
		},
		{
			name:  "a ProductGroup's variants carry its offers",
			draft: `{"@type":"ProductGroup","name":"Box","productGroupID":"box","variesBy":"https://schema.org/size","hasVariant":[{"@type":"Product","name":"Box, small","size":"small","offers":{"@type":"Offer","price":1,"priceCurrency":"USD","availability":"InStock"},"review":{"@type":"Review","author":{"@type":"Person","name":"Ann Lee"}},"aggregateRating":{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}}]}`,
		},
		{
			name:  "a reference is checked where its node is",
			draft: `{"@type":"WebPage","name":"Boxes","url":"https://example.com/boxes/","about":{"@type":"Product","@id":"https://example.com/#box"}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issues := snippetIssues(t, tc.draft)
			var got []string
			for _, i := range issues {
				msg := strings.TrimPrefix(i.Message, "Search Console (Product snippets), ")
				got = append(got, i.Path+" "+msg)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("%d findings, want %d:\n%s", len(got), len(tc.want), strings.Join(got, "\n"))
			}
			for i, w := range tc.want {
				if !regexp.MustCompile(w).MatchString(got[i]) {
					t.Errorf("finding %d\n got %s\nwant /%s/", i, got[i], w)
				}
			}
		})
	}
}
