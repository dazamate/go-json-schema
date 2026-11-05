// Package product holds the product checks: they hold a shop's markup to
// what its pages show of the products. A product page's own checks cover the
// Product itself, its choices of size or colour, its details, sizes, sets and
// audience. The checks that run on any page of a shop cover its offers (sale
// and was prices, unit prices, surcharges, add-ons, stock, condition, sale
// dates, member prices), the reviews, share and wishlist buttons and the
// order tracking it shows, and when and by whom the business was founded.
// On another site the same words ("a 1.5% surcharge", "save 20%") are about
// others, so the offer checks run only on a shop's pages.
package product

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	wasText = regexp.MustCompile(`(?i)\b(?:was|rrp|originally|regular price|compare at)\s*:?\s*((?:A\$|AU\$|\$|£|€)\s?\d[\d,]*(?:\.\d{1,2})?)`)
)

// wasWords are the class words that mark an old price, and wasCompounds the
// whole class tokens that do so with their hyphen.
var (
	wasWords     = []string{"compare", "original", "was", "old", "regular", "strike", "strikethrough", "rrp", "crossed"}
	wasCompounds = []string{"line-through", "list-price", "compare-at"}
)

// wasClass reports whether an attribute value (a class list, an id) names an
// old price: one of its whole class tokens is a was compound, or one of its
// words, split on hyphens, underscores and camelCase, is a was word. "font-bold"
// and "holder" hold "old" but no word of it.
func wasClass(v string) bool {
	for _, tok := range strings.Fields(strings.ToLower(v)) {
		if slices.Contains(wasCompounds, tok) {
			return true
		}
	}
	words := strings.FieldsFunc(camelSplit.ReplaceAllString(v, "$1 $2"), func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	for _, w := range words {
		if slices.Contains(wasWords, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

var camelSplit = regexp.MustCompile(`([a-z])([A-Z])`)

// sale is what a shop's page shows of a sale.
type sale struct {
	was     []string // the was prices the page shows
	struck  []string // the prices the draft's StrikethroughPrice and ListPrice specifications carry
	missing []string // the was prices none of those carries
}

// saleKey is the memo key of the page's sale.
type saleKey struct{}

// saleFound is the memoized saleOf: the sale, and whether the page is a
// shop's.
type saleFound struct {
	s  sale
	ok bool
}

// saleOf is the sale a page search engines index of a shop shows, worked out
// once for the checks that share it; ok is false on any other page.
func saleOf(c *audit.Context) (s sale, ok bool) {
	f := audit.Memo(c, saleKey{}, func() saleFound {
		s, ok := findSale(c)
		return saleFound{s, ok}
	})
	return f.s, f.ok
}

// findSale reads the sale off the page and the draft.
func findSale(c *audit.Context) (s sale, ok bool) {
	if !c.Indexed() || !c.Page.Sells() {
		return sale{}, false
	}
	s.was = wasPrices(c.Page)
	for _, l := range c.Nodes {
		if c.Is(l.N, "PriceSpecification") && slices.ContainsFunc(jsonld.Strs(l.N, "priceType"), func(t string) bool {
			return jsonld.StripPrefix(t) == "StrikethroughPrice" || jsonld.StripPrefix(t) == "ListPrice"
		}) {
			s.struck = append(s.struck, jsonld.Scalars(l.N, "price")...)
		}
	}
	for _, w := range s.was {
		m := txt.PriceAmount.FindStringSubmatch(w)
		if f, ok := txt.PriceOf(m); ok && !txt.HoldsNumber(s.struck, f) {
			s.missing = append(s.missing, w)
		}
	}
	return s, true
}

// wasPrices are the was prices the page shows outside its header, footer and
// navs: struck through, in an element marked as the old price, or written
// after "was" or "RRP".
func wasPrices(p *webpage.Facts) []string {
	var out []string
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	if p.Doc != nil {
		webpage.Visit(p.Doc, func(n *html.Node) {
			if webpage.Chrome(n) {
				return
			}
			t := webpage.HTMLText(n)
			ms := txt.PriceAmount.FindAllString(t, -1)
			if len(ms) != 1 || len(strings.Fields(t)) > 3 {
				return
			}
			switch n.DataAtom {
			case atom.S, atom.Del, atom.Strike:
				add(ms[0])
				return
			}
			for _, k := range []string{"class", "data-testid", "itemprop", "id"} {
				if v, _ := webpage.Attr(n, k); v != "" && wasClass(v) {
					add(ms[0])
					return
				}
			}
		})
	}
	for _, m := range wasText.FindAllStringSubmatch(p.Main, -1) {
		add(m[1])
	}
	return out
}
