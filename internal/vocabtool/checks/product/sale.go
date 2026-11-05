// Package product holds the offer checks: what a shop's pages say about its
// own sales that the Offers must carry. On another site the same words ("a
// 1.5% surcharge", "save 20%") are about others, so the checks run only on a
// shop's pages.
package product

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	wasClass = regexp.MustCompile(`(?i)compare|original|was|old|regular|strike|line-through|rrp|list-price|crossed`)
	wasText  = regexp.MustCompile(`(?i)\b(?:was|rrp|originally|regular price|compare at)\s*:?\s*((?:A\$|AU\$|\$|£|€)\s?\d[\d,]*(?:\.\d{1,2})?)`)
)

// sale is what a shop's page shows of a sale.
type sale struct {
	was     []string // the was prices the page shows
	struck  []string // the prices the draft's StrikethroughPrice and ListPrice specifications carry
	missing []string // the was prices none of those carries
}

// saleOf is the sale a page search engines index of a shop shows; ok is
// false on any other page.
func saleOf(c *audit.Context) (s sale, ok bool) {
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
				if v, _ := webpage.Attr(n, k); v != "" && wasClass.MatchString(v) {
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
