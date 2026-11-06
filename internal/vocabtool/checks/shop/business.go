package shop

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// storeKinds name the Store subtype a shopfront business takes from the words
// its name uses, most specific first.
var storeKinds = []struct {
	words *regexp.Regexp
	use   string
}{
	{regexp.MustCompile(`(?i)\b(menswear|men'?s (clothing|wear|fashion))\b`), "MensClothingStore"},
	{regexp.MustCompile(`(?i)\b(jewell?e?ry|jewell?ers?)\b`), "JewelryStore"},
	{regexp.MustCompile(`(?i)\b(auto|car|motor) parts\b`), "AutoPartsStore"},
	{regexp.MustCompile(`(?i)\b(bikes?|bicycles?|cycles?)\b`), "BikeStore"},
	{regexp.MustCompile(`(?i)\b(books?|bookshop|bookstore|booksellers?)\b`), "BookStore"},
	{regexp.MustCompile(`(?i)\b(computers?|laptops?)\b`), "ComputerStore"},
	{regexp.MustCompile(`(?i)\bconvenience\b`), "ConvenienceStore"},
	{regexp.MustCompile(`(?i)\bdepartment stores?\b`), "DepartmentStore"},
	{regexp.MustCompile(`(?i)\belectronics?\b`), "ElectronicsStore"},
	{regexp.MustCompile(`(?i)\b(florists?|flowers)\b`), "Florist"},
	{regexp.MustCompile(`(?i)\bfurniture\b`), "FurnitureStore"},
	{regexp.MustCompile(`(?i)\b(garden|nursery)\b`), "GardenStore"},
	{regexp.MustCompile(`(?i)\b(grocery|grocers?|supermarket)\b`), "GroceryStore"},
	{regexp.MustCompile(`(?i)\bhardware\b`), "HardwareStore"},
	{regexp.MustCompile(`(?i)\b(hobby|hobbies|crafts?)\b`), "HobbyShop"},
	{regexp.MustCompile(`(?i)\b(homewares?|home goods)\b`), "HomeGoodsStore"},
	{regexp.MustCompile(`(?i)\b(liquor|bottle shop|wine shop)\b`), "LiquorStore"},
	{regexp.MustCompile(`(?i)\b(mobile phones?|phones)\b`), "MobilePhoneStore"},
	{regexp.MustCompile(`(?i)\b(movie|video) rentals?\b`), "MovieRentalStore"},
	{regexp.MustCompile(`(?i)\b(music|records|instruments)\b`), "MusicStore"},
	{regexp.MustCompile(`(?i)\boffice (supplies|equipment)\b|\bstationery\b`), "OfficeEquipmentStore"},
	{regexp.MustCompile(`(?i)\boutlet\b`), "OutletStore"},
	{regexp.MustCompile(`(?i)\bpawn(brokers?|shop)?\b`), "PawnShop"},
	{regexp.MustCompile(`(?i)\bpets?\b`), "PetStore"},
	{regexp.MustCompile(`(?i)\b(shoes|footwear)\b`), "ShoeStore"},
	{regexp.MustCompile(`(?i)\b(sports?|sporting goods)\b`), "SportingGoodsStore"},
	{regexp.MustCompile(`(?i)\b(tyres?|tires?)\b`), "TireShop"},
	{regexp.MustCompile(`(?i)\btoys?\b`), "ToyStore"},
	{regexp.MustCompile(`(?i)\bwholesale\b`), "WholesaleStore"},
	{regexp.MustCompile(`(?i)\b(clothing|fashion|apparel|boutique)\b`), "ClothingStore"},
}

// copyrightOwner finds who a © line names: the text after the © and any
// years.
var copyrightOwner = regexp.MustCompile(`©\s*(?:\d{4}\s*[-–]?\s*)*([^·|\n]{2,60})`)

// businessName is what the page calls the business: its og:site_name, its
// copyright holder, and its title.
func businessName(p *webpage.Facts) string {
	var parts []string
	if p.Doc != nil {
		parts = append(parts, webpage.MetaContent(p.Doc, "property", "og:site_name"))
		for _, t := range webpage.QueryAll(p.Doc, "title") {
			parts = append(parts, webpage.HTMLText(t))
		}
	}
	if m := copyrightOwner.FindStringSubmatch(p.Text); m != nil {
		parts = append(parts, m[1])
	}
	return strings.Join(parts, " · ")
}

// storeType is the Store subtype for the business's trade, or Store.
func storeType(p *webpage.Facts) string {
	name := businessName(p) + " · " + p.H1()
	for _, k := range storeKinds {
		if k.words.MatchString(name) {
			return k.use
		}
	}
	return "Store"
}

// shopBusiness is the business a shop's page calls for and the draft's node
// for it.
type shopBusiness struct {
	want string      // the type the business takes: the Store subtype for its trade, or OnlineStore
	why  string      // what on the page calls for that type
	node jsonld.Node // the draft's first Organization with properties; N is nil when it has none
}

// shopBusinessKey is the memo key of the shop's business.
type shopBusinessKey struct{}

// shopBusinessOf is the business the page calls for, worked out once: a shop
// that sells online is an OnlineStore, and one whose page shows a street
// address the Store subtype for its trade, even when it also sells online.
// It is nil on a page search engines skip, one known only by its text, and
// one that neither sells nor shows a street address.
func shopBusinessOf(c *audit.Context) *shopBusiness {
	return audit.Memo(c, shopBusinessKey{}, func() *shopBusiness {
		if !c.Indexed() || c.Page.Doc == nil {
			return nil
		}
		b := &shopBusiness{}
		switch addr := c.Page.Address(); {
		case addr != "":
			b.want = storeType(c.Page)
			b.why = fmt.Sprintf("the page shows the street address %q", txt.Clip(addr, 60))
		case c.Page.Sells():
			b.want, b.why = "OnlineStore", "the site sells online"
		default:
			return nil
		}
		if i := slices.IndexFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "Organization") && len(l.N.Keys) > 1 }); i >= 0 {
			b.node = c.Nodes[i]
		}
		return b
	})
}
