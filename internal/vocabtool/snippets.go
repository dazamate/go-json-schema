package vocabtool

import (
	"fmt"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// The product snippet checks hold every Product node to Google's Product
// snippet requirements
// (https://developers.google.com/search/docs/appearance/structured-data/product-snippet,
// last updated 2026-09-08). Search Console's Product snippets report reads a
// subtype of Product as a Product: a Vehicle, ProductModel or
// IndividualProduct is held to the same rules. A ProductGroup is not: its
// variants carry the offers, and each variant is a Product checked itself.
//
// Each message quotes the report's own wording, so a finding can be matched to
// the issue Search Console lists:
//   - critical, which marks the item invalid there, is a warning here: the node
//     is still valid schema.org, and only vocabulary errors are errors;
//   - non-critical, which the report only warns on, is info here: a
//     recommended field such as availability, offerCount or review takes a fact
//     the page or its sources may not give, and is never to be invented. The
//     content checks already ask for what the page shows (its prices, stock and
//     ratings).

const (
	snippetCritical    = `Search Console (Product snippets), critical: `
	snippetNonCritical = `Search Console (Product snippets), non-critical: `
)

// productSnippets checks each Product node and its offers. A bare reference to
// a node is checked where the node itself is.
func (c *checker) productSnippets(all []jsonld.Node) {
	byID := map[string]*jsonld.Object{}
	for _, l := range all {
		if jsonld.Described(l.N) {
			for _, id := range jsonld.Strs(l.N, "@id") {
				if _, seen := byID[id]; !seen {
					byID[id] = l.N
				}
			}
		}
	}
	// resolve is the node a value is, following a bare {"@id"} reference.
	resolve := func(v any) *jsonld.Object {
		o, ok := v.(*jsonld.Object)
		if !ok {
			return nil
		}
		if !jsonld.Described(o) {
			for _, id := range jsonld.Strs(o, "@id") {
				if n, ok := byID[id]; ok {
					return n
				}
			}
		}
		return o
	}
	for _, l := range all {
		if !jsonld.Described(l.N) || !c.is(l.N, "Product") {
			continue
		}
		ty := jsonld.TypeNames(l.N)[0]
		if _, ok := l.N.Get("name"); !ok {
			c.add(l.Path, Warning, snippetCritical+`Missing field "name": give this %s the name the page shows for it`, ty)
		}
		if c.is(l.N, "ProductGroup") {
			c.snippetOffers(l, resolve)
			continue
		}
		var missing []string
		for _, k := range []string{"offers", "review", "aggregateRating"} {
			if _, ok := l.N.Get(k); !ok {
				missing = append(missing, k)
			}
		}
		switch len(missing) {
		case 3:
			kind := "a Product"
			if ty != "Product" {
				kind = "a " + ty + ", a kind of Product, so the report reads it as one"
			}
			c.add(l.Path, Warning, snippetCritical+`Either "offers", "review", or "aggregateRating" should be specified. This node is %s, and the item is invalid. Give it offers with the price the page shows for it (an Offer, or an AggregateOffer for a range), or review or aggregateRating only from real reviews and ratings the page shows. If the page shows none of them, it is not a product snippet: show the price, or type the node outside Product`, kind)
		case 0:
		default:
			var fields []string
			for _, k := range missing {
				fields = append(fields, fmt.Sprintf(`Missing field %q`, k))
			}
			advice := ". Add offers only with the price the page shows"
			if _, ok := l.N.Get("offers"); ok {
				advice = `. Add review and aggregateRating only from real reviews and ratings the page shows, never invented ones: fake ratings break Google's review snippet guidelines and can bring a manual action`
			}
			c.add(l.Path, Info, "%s%s%s", snippetNonCritical, strings.Join(fields, "; "), advice)
		}
		c.snippetOffers(l, resolve)
	}
}

// snippetOffers checks each of a Product node's offers.
func (c *checker) snippetOffers(l jsonld.Node, resolve func(any) *jsonld.Object) {
	raw, _ := l.N.Get("offers")
	for i, v := range jsonld.Values(raw) {
		path := l.Path + ".offers"
		if _, list := raw.([]any); list {
			path = fmt.Sprintf("%s[%d]", path, i)
		}
		if o := resolve(v); o != nil {
			c.snippetOffer(path, o, resolve)
		}
	}
}

// snippetOffer checks one of a Product's offers: an AggregateOffer's price
// range, or an Offer's price, and the price specifications it nests.
func (c *checker) snippetOffer(path string, o *jsonld.Object, resolve func(any) *jsonld.Object) {
	has := func(n *jsonld.Object, k string) bool { _, ok := n.Get(k); return ok }
	var specs []*jsonld.Object
	raw, _ := o.Get("priceSpecification")
	for _, v := range jsonld.Values(raw) {
		if s := resolve(v); s != nil {
			specs = append(specs, s)
		}
	}
	specHas := func(k string) bool {
		for _, s := range specs {
			if has(s, k) {
				return true
			}
		}
		return false
	}
	if c.is(o, "AggregateOffer") {
		for _, k := range []string{"lowPrice", "priceCurrency"} {
			if !has(o, k) {
				c.add(path, Warning, snippetCritical+`Missing field %q (in "offers"): an AggregateOffer needs lowPrice and priceCurrency`, k)
			}
		}
		for _, k := range []string{"highPrice", "offerCount"} {
			if !has(o, k) {
				what := "the highest price the range reaches"
				if k == "offerCount" {
					what = "how many offers the range spans: the sellers or sources it comes from"
				}
				c.add(path, Info, snippetNonCritical+`Missing field %q (in "offers"): give the AggregateOffer %s, %s`, k, k, what)
			}
		}
	} else if c.is(o, "Offer") {
		if !has(o, "price") && !specHas("price") {
			c.add(path, Warning, snippetCritical+`Missing field "price" (in "offers"): give the Offer price, or a priceSpecification with price`)
		}
		if !has(o, "priceCurrency") && !specHas("priceCurrency") {
			c.add(path, Info, snippetNonCritical+`Missing field "priceCurrency" (in "offers"): give the Offer's currency as a three-letter ISO 4217 code`)
		}
		if !has(o, "availability") {
			c.add(path, Info, snippetNonCritical+`Missing field "availability" (in "offers"): give the Offer's availability, an ItemAvailability member such as InStock`)
		}
	}
	for i, s := range specs {
		if c.is(s, "PriceSpecification") && !has(s, "price") {
			p := path + ".priceSpecification"
			if _, list := raw.([]any); list {
				p = fmt.Sprintf("%s[%d]", p, i)
			}
			c.add(p, Warning, snippetCritical+`Missing field "price" (in "offers.priceSpecification"): Google requires price on a %s under a product's offers, and minPrice and maxPrice do not stand in for it. Give it price, or carry a range on the offer alone (an AggregateOffer's lowPrice and highPrice)`, jsonld.TypeNames(s)[0])
		}
	}
}
