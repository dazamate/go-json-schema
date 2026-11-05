// Package snippets holds the product snippet checks. They hold every Product
// node to Google's Product snippet requirements
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
//
// A bare reference to a node is checked where the node itself is. The checks
// run on the draft alone, with or without a page.
package snippets

// The prefixes of every message, Search Console's own wording.
const (
	critical    = `Search Console (Product snippets), critical: `
	nonCritical = `Search Console (Product snippets), non-critical: `
)
