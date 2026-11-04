package vocabtool

import (
	"fmt"
	"strings"
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
func (c *checker) productSnippets(all []located) {
	byID := map[string]*object{}
	for _, l := range all {
		if described(l.n) {
			for _, id := range strs(l.n, "@id") {
				if _, seen := byID[id]; !seen {
					byID[id] = l.n
				}
			}
		}
	}
	// resolve is the node a value is, following a bare {"@id"} reference.
	resolve := func(v any) *object {
		o, ok := v.(*object)
		if !ok {
			return nil
		}
		if !described(o) {
			for _, id := range strs(o, "@id") {
				if n, ok := byID[id]; ok {
					return n
				}
			}
		}
		return o
	}
	for _, l := range all {
		if !described(l.n) || !c.is(l.n, "Product") {
			continue
		}
		ty := typeNames(l.n)[0]
		if _, ok := l.n.get("name"); !ok {
			c.add(l.path, Warning, snippetCritical+`Missing field "name": give this %s the name the page shows for it`, ty)
		}
		if c.is(l.n, "ProductGroup") {
			c.snippetOffers(l, resolve)
			continue
		}
		var missing []string
		for _, k := range []string{"offers", "review", "aggregateRating"} {
			if _, ok := l.n.get(k); !ok {
				missing = append(missing, k)
			}
		}
		switch len(missing) {
		case 3:
			kind := "a Product"
			if ty != "Product" {
				kind = "a " + ty + ", a kind of Product, so the report reads it as one"
			}
			c.add(l.path, Warning, snippetCritical+`Either "offers", "review", or "aggregateRating" should be specified. This node is %s, and the item is invalid. Give it offers with the price the page shows for it (an Offer, or an AggregateOffer for a range), or review or aggregateRating only from real reviews and ratings the page shows. If the page shows none of them, it is not a product snippet: show the price, or type the node outside Product`, kind)
		case 0:
		default:
			var fields []string
			for _, k := range missing {
				fields = append(fields, fmt.Sprintf(`Missing field %q`, k))
			}
			advice := ". Add offers only with the price the page shows"
			if _, ok := l.n.get("offers"); ok {
				advice = `. Add review and aggregateRating only from real reviews and ratings the page shows, never invented ones: fake ratings break Google's review snippet guidelines and can bring a manual action`
			}
			c.add(l.path, Info, "%s%s%s", snippetNonCritical, strings.Join(fields, "; "), advice)
		}
		c.snippetOffers(l, resolve)
	}
}

// snippetOffers checks each of a Product node's offers.
func (c *checker) snippetOffers(l located, resolve func(any) *object) {
	raw, _ := l.n.get("offers")
	for i, v := range values(raw) {
		path := l.path + ".offers"
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
func (c *checker) snippetOffer(path string, o *object, resolve func(any) *object) {
	has := func(n *object, k string) bool { _, ok := n.get(k); return ok }
	var specs []*object
	raw, _ := o.get("priceSpecification")
	for _, v := range values(raw) {
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
			c.add(p, Warning, snippetCritical+`Missing field "price" (in "offers.priceSpecification"): Google requires price on a %s under a product's offers, and minPrice and maxPrice do not stand in for it. Give it price, or carry a range on the offer alone (an AggregateOffer's lowPrice and highPrice)`, typeNames(s)[0])
		}
	}
}

// described reports a node that says something itself, not only a reference
// to a node elsewhere: a property beyond @id and @type.
func described(n *object) bool {
	for _, k := range n.keys {
		if !strings.HasPrefix(k, "@") {
			return true
		}
	}
	return false
}
