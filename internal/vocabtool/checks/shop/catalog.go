package shop

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Catalog{}) }

var categorySegment = regexp.MustCompile(`^(categor(y|ies)|collections?|c|departments?)$`)

// Catalog warns when the site's menus list its categories and no node has an
// OfferCatalog of them: the business's hasOfferCatalog names the categories
// it sells in. It needs the page's document, a shop's page and at least three
// categories.
type Catalog struct{}

// ID is the check's ID.
func (Catalog) ID() string { return "shop/catalog" }

// Run reports the categories the menus link, once, up to six.
func (Catalog) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || !c.Page.Sells() || jsonld.HasKey(c.Nodes, "hasOfferCatalog") {
		return nil
	}
	p := c.Page
	var names, seen []string
	for _, nav := range webpage.QueryAll(p.Doc, "nav, header") {
		for _, a := range webpage.QueryAll(nav, "a[href]") {
			u, ok := p.SiteLink(a)
			if !ok || slices.Contains(seen, u) {
				continue
			}
			pu, _ := url.Parse(u)
			segs := webpage.Segments(pu)
			if i := slices.IndexFunc(segs, categorySegment.MatchString); i >= 0 && i < len(segs)-1 && pu.RawQuery == "" {
				seen = append(seen, u)
				names = append(names, strings.TrimSpace(webpage.HTMLText(a)))
			}
		}
	}
	if len(seen) < 3 {
		return nil
	}
	list := names
	if len(list) > 6 {
		list = append(list[:6:6], "…")
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the site's menu lists %d categories (%s): set the business's hasOfferCatalog, an OfferCatalog with a ListItem for each category naming its CollectionPage", len(seen), strings.Join(list, ", "))}
}
