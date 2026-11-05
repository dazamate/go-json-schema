package shop

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
)

func init() { audit.Register(Listing{}) }

// Listing warns on the product cards a page lists that no ItemList carries.
// A listing's products are its subject; a row of them elsewhere, such as the
// home page's best sellers, is a part of the page. It takes the prices of
// every card and of the page's filter links and starting price, so the
// prices checks do not ask for them as items' prices.
type Listing struct{}

// ID is the check's ID.
func (Listing) ID() string { return "shop/listing" }

// Run reports the listed products whose address no value in the draft holds.
func (Listing) Run(c *audit.Context) []audit.Issue {
	cards := listed(c)
	if cards == nil {
		return nil
	}
	d := c.Index()
	var missing []string
	for _, cd := range cards {
		if !d.URLs[cd.URL] && !slices.Contains(missing, cd.Name) {
			missing = append(missing, cd.Name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	where := "the WebPage's hasPart (name its heading)"
	if c.Page.Kind == webpage.KindListing {
		where = "the CollectionPage's mainEntity (numberOfItems, itemListOrder)"
	}
	names := missing
	if len(names) > 5 {
		names = append(names[:5:5], "…")
	}
	return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, commonParent(cards)), audit.Warning, "the page lists %d products (%s) that no ItemList carries: add an ItemList as %s, with a ListItem for each product (position, url) holding its Product (name, url, image, offers an Offer with price and priceCurrency)", len(missing), strings.Join(names, ", "), where)}
}

// TakesPrices takes the prices on every product card, related ones too,
// whether or not an ItemList carries them; the prices in the page's filter
// links; and the listing's starting price.
func (Listing) TakesPrices(c *audit.Context, _ audit.Taken) audit.Taken {
	if !c.Indexed() {
		return nil
	}
	p := c.Page
	t := audit.Taken{}
	// a filter link (Under $10, $20 and over) names no item's price: its
	// address holds a query, and its text is little besides the price
	if p.Doc != nil {
		for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
			if u, ok := p.SiteLink(a); ok && webpage.MustParse(u).RawQuery != "" {
				if text := webpage.HTMLText(a); len(strings.Fields(txt.PriceAmount.ReplaceAllString(text, ""))) <= 3 {
					t.Take(text)
				}
			}
		}
	}
	for _, cd := range p.ProductCards() {
		t.Take(webpage.HTMLText(cd.N))
	}
	if m := startingPrice(c); m != nil {
		t.Take(m[0])
	}
	return t
}

// commonParent is the nearest element around every card.
func commonParent(cards []*webpage.Card) *html.Node {
	n := cards[0].N
	for n != nil {
		if !slices.ContainsFunc(cards, func(cd *webpage.Card) bool {
			for e := cd.N; e != nil; e = e.Parent {
				if e == n {
					return false
				}
			}
			return true
		}) {
			return n
		}
		n = n.Parent
	}
	return cards[0].N
}
