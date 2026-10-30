package vocabtool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// checkShop checks a draft against a fixture under testdata/shop, served at
// the fixture's address.
func checkShop(t *testing.T, key, draft string) Report {
	t.Helper()
	p := shopPages[key]
	b, err := os.ReadFile(filepath.Join("testdata", "shop", p.file))
	if err != nil {
		t.Fatal(err)
	}
	if draft == "" {
		draft = `{"@context":"https://schema.org","@graph":[]}`
	}
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{PageHTML: string(b), PageURL: p.url})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return r
}

// fixtureDraft is the markup a fixture's page carries now.
func fixtureDraft(t *testing.T, key string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "shop", strings.TrimSuffix(shopPages[key].file, ".html")+".jsonld"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// warns reports a warning, anywhere, holding every one of the parts.
func warns(r Report, parts ...string) bool {
	return slices.ContainsFunc(r.Issues, func(i Issue) bool {
		return i.Severity == Warning && !slices.ContainsFunc(parts, func(p string) bool { return !strings.Contains(i.Message, p) })
	})
}

func expectWarnings(t *testing.T, r Report, want [][]string) {
	t.Helper()
	for _, parts := range want {
		if !warns(r, parts...) {
			t.Errorf("no warning holding %q", parts)
		}
	}
	if t.Failed() {
		for _, i := range r.Issues {
			t.Logf("%s %s: %s", i.Severity, i.Path, i.Message)
		}
	}
}

func TestCheckKeepsGoogleActionAnnotations(t *testing.T) {
	r := check(t, `{"@type":"WebSite","url":"https://x.test/","potentialAction":{"@type":"SearchAction",
		"target":{"@type":"EntryPoint","urlTemplate":"https://x.test/search?q={search_term_string}"},"query-input":"required name=search_term_string"}}`)
	if r.Errors != 0 {
		t.Errorf("errors on query-input: %+v", r.Issues)
	}
	if !issueAt(r, Info, "$.potentialAction.query-input", "Google's action annotation") {
		t.Errorf("no info on query-input: %+v", r.Issues)
	}
}

func TestCheckLeavesAGalleryCounterOutOfRatings(t *testing.T) {
	page := `<!doctype html><html><head><link rel="canonical" href="https://x.test/p/ring"></head><body><main>
<div class="gallery"><span>1 / 10</span></div><p>Rated 4.5/5 by buyers. Scores 8/10 on the review site.</p></main></body></html>`
	r := checkPage(t, `{"@context":"https://schema.org","@graph":[]}`, page)
	if !warns(r, "shows ratings (4.5, 8)") {
		t.Errorf("want the ratings 4.5 and 8 only: %+v", r.Issues)
	}
}

func TestCheckLeavesProductCardTitlesOutOfCounts(t *testing.T) {
	r := checkShop(t, "qod-product-set", "")
	if n := issueCount(r, "counts 2 items"); n != 0 {
		t.Errorf("%d count warnings on the set's own name: %+v", n, r.Issues)
	}
}

func TestCheckReadsAKeyValueListAsData(t *testing.T) {
	r := checkShop(t, "qod-shipping", "")
	if n := issueCount(r, "DefinedTerm names"); n != 0 {
		t.Errorf("the shipping page's key/value list became DefinedTerms: %+v", r.Issues)
	}
	glossary := `<!doctype html><html><body><main><dl>
<dt>Bezel</dt><dd>A rim of metal that holds a stone in its setting.</dd>
<dt>Prong</dt><dd>A thin claw of metal that grips the stone at its edge.</dd></dl></main></body></html>`
	r = checkPage(t, `{"@context":"https://schema.org","@graph":[]}`, glossary)
	if !warns(r, "defines 2 terms (Bezel, Prong)") {
		t.Errorf("a glossary is no longer DefinedTerms: %+v", r.Issues)
	}
	// a shop's glossary is one too; its dl that sells a category is not
	shop := `<!doctype html><html><body><header><a href="/">Gems</a><a href="/cart">Bag</a></header><main><h1>Glossary</h1>
<dl><dt>Bezel</dt><dd>A rim of metal that holds a stone in its setting.</dd>
<dt>Prong</dt><dd>A thin claw of metal that grips the stone at its edge.</dd></dl>
<section><h2>A locket that holds a little more</h2><div><dl><dt>Harmony chime lockets</dt><dd>A small chime ball inside rings softly as you move.</dd>
<dt>Diffuser lockets</dt><dd>Add a few drops of your favourite oil to a pad, and the scent stays with you all day.</dd></dl>
<a href="/collections/lockets">Shop lockets</a></div></section></main></body></html>`
	r = checkShopPage(t, shop, "https://gems.test/pages/glossary")
	if !warns(r, "defines 2 terms (Bezel, Prong)") {
		t.Errorf("want the glossary's terms and not the lockets: %+v", r.Issues)
	}
}

func TestCheckLeavesMenuPricesOutOfPrices(t *testing.T) {
	r := checkShop(t, "qod-category", "")
	for _, i := range r.Issues {
		if strings.Contains(i.Message, "that no price property carries") && (strings.Contains(i.Message, "$10,") || strings.Contains(i.Message, "$20,")) {
			t.Errorf("menu filter prices reported: %s", i.Message)
		}
	}
}

func TestCheckFindsTheSiteHeaderAndFooterInsideMain(t *testing.T) {
	r := checkShop(t, "qod-terms", "")
	var types []string
	for _, e := range r.PageElements {
		types = append(types, e.Type)
	}
	for _, want := range []string{"WPHeader", "WPFooter"} {
		if !slices.Contains(types, want) {
			t.Errorf("page_elements %v lack %s", types, want)
		}
	}
	// the category page's own header, which holds its h1, is not the site's
	r = checkShop(t, "qod-category", "")
	n := 0
	for _, e := range r.PageElements {
		if e.Type == "WPHeader" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d WPHeaders on the category page, want 1: %+v", n, r.PageElements)
	}
}

func TestCheckSkipsTheShopChecksOnANoindexPage(t *testing.T) {
	r := checkShop(t, "qod-search", "")
	if !slices.ContainsFunc(r.Issues, func(i Issue) bool {
		return i.Severity == Info && strings.Contains(i.Message, "noindex") && strings.Contains(i.Message, "SearchResultsPage")
	}) {
		t.Errorf("no noindex info: %+v", r.Issues)
	}
	if r.Warnings != 0 {
		t.Errorf("%d warnings on a noindex page: %+v", r.Warnings, r.Issues)
	}
}

// checkShopPage checks a draft against a page served at an address.
func checkShopPage(t *testing.T, page, address string) Report {
	t.Helper()
	r, err := newTool(t).Check(json.RawMessage(`{"@context":"https://schema.org","@graph":[]}`), CheckOptions{PageHTML: page, PageURL: address})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return r
}

func TestCheckLeavesShopChromeOutOfItsFindings(t *testing.T) {
	r := checkShop(t, "qod-home", "")
	for _, i := range r.Issues {
		if strings.Contains(i.Message, "lists 8 products") && strings.Contains(i.Message, "$") {
			t.Errorf("product names carry their card's price: %s", i.Message)
		}
		if strings.Contains(i.Message, "numberOfItems 1153") {
			t.Errorf("a category tile's count became the home page's: %s", i.Message)
		}
	}
	// "Free over $60, otherwise $7.95" is the shipping rate and threshold;
	// a row of featured products is a part of the home page, not its subject
	expectWarnings(t, r, [][]string{{"free shipping over $60", "orderValue"}, {"lists 8 products", "WebPage's hasPart"}})

	listing := `<!doctype html><html><body><main><h1>Chakra bracelets</h1>
<nav><ul><li><a href="/au/store?price=10-15">$10 to $15</a></li><li><a href="/au/store?price=20-plus">$20 and over</a></li>
<li><a href="/au/categories/chakra?price=under-10">Under $10</a></li></ul></nav>
<p>Most pieces are under $20, so three or four together ship free.</p>
<ul><li><a href="/au/products/a"><h3>Seven Chakra Bracelet</h3><span>$12.95</span></a></li>
<li><a href="/au/products/b"><h3>Lava Stone Bracelet</h3><span>$9.95</span></a></li></ul></main></body></html>`
	r = checkShopPage(t, listing, "http://localhost:8001/au/categories/chakra")
	for _, i := range r.Issues {
		if strings.Contains(i.Message, "that no price property carries") {
			t.Errorf("filter links or a price bound read as prices: %s", i.Message)
		}
	}

	product := `<!doctype html><html><body><main><h1>Cheshire Cat Locket</h1>
<form action="/au/cart/add" method="post"><button>Add to bag</button></form>
<dl><dt>Pendant Material</dt><dd>Silver Plated/ Glass Chain Material: 925 Sterling Silver Filled Pendant Locket Size: 2.5cm x 2.5cm Chain Length: 50cm Includes 4 Oil Diffuser Pads</dd></dl>
<dl><dt>Weight</dt><dd>-</dd><dt>Finish</dt><dd>Polished</dd><dt>Clasp</dt><dd>Lobster clasp with a short extender chain.</dd></dl>
</main></body></html>`
	r = checkShopPage(t, product, "http://localhost:8001/au/products/cheshire-cat-locket")
	if n := issueCount(r, "DefinedTerm"); n != 0 {
		t.Errorf("a product's details became DefinedTerms: %+v", r.Issues)
	}
	expectWarnings(t, r, [][]string{{"Pendant Material", "material"}, {"Finish", "Clasp"}})
	if n := issueCount(r, `Weight "-"`); n != 0 {
		t.Errorf("an empty detail is reported: %+v", r.Issues)
	}
}

func TestCheckHoldsAProductDetailToItsWholeValue(t *testing.T) {
	page := `<!doctype html><html><body><main><h1>Moon Locket</h1>
<form action="/au/cart/add" method="post"><button>Add to bag</button></form>
<ul><li>Chain Material: 925 Sterling Silver Filled</li><li>Locket Size: 2.5cm x 2.5cm</li></ul></main></body></html>`
	draft := func(material, size string) string {
		return `{"@context":"https://schema.org","@type":"Product","name":"Moon Locket","material":"` + material + `","size":"` + size + `"}`
	}
	check := func(d string) Report {
		r, err := newTool(t).Check(json.RawMessage(d), CheckOptions{PageHTML: page, PageURL: "http://localhost:8001/au/products/moon-locket"})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if r := check(draft("925 Sterling Silver Filled", "2.5cm x 2.5cm")); issueCount(r, "product details the markup lacks") != 0 {
		t.Errorf("the details as the page gives them are reported: %+v", r.Issues)
	}
	// "Silver" drops the 925 the page gives
	if r := check(draft("Silver", "2.5 x 2.5 cm")); !warns(r, "Chain Material", "material") || warns(r, "Locket Size") {
		t.Errorf("want Chain Material reported, Locket Size not: %+v", r.Issues)
	}
}

func TestCheckFindsWhatAProductPageShows(t *testing.T) {
	r := checkShop(t, "qod-product-sale", fixtureDraft(t, "qod-product-sale"))
	expectWarnings(t, r, [][]string{
		{"$15.95", "was price", "StrikethroughPrice"},
		{"GST included", "valueAddedTaxIncluded"},
		{"free shipping over $60", "orderValue"},
		{"add-to-bag form", "BuyAction", "http://localhost:8001/au/cart/add"},
		{"Chain Length", "additionalProperty"},
		{"Chain Material", "material"},
		{"no WebSite"},
		{"OnlineStore"},
		{"ItemPage"},
		{"links the return policy", "merchantReturnLink"},
		{"instagram.com/queen_of_diamonds_jewellery", "sameAs"},
		{"You may also like", "isRelatedTo"},
	})
	// the sale's was price and the related products' prices are each
	// reported once, by their own check
	for _, i := range r.Issues {
		if strings.Contains(i.Message, "that no price property carries") {
			t.Errorf("leftover prices: %s", i.Message)
		}
	}
}

func TestCheckFindsAProductsChoices(t *testing.T) {
	r := checkShop(t, "qod-product-variants", fixtureDraft(t, "qod-product-variants"))
	expectWarnings(t, r, [][]string{
		{`variesBy "Size"`, "https://schema.org/size"},
		{"this variant has no size", `"6"`},
		{"6: 16mm", "SizeSpecification"},
	})
	r = checkShop(t, "qod-product-variants", "")
	expectWarnings(t, r, [][]string{{"a choice of Size (6, 7, 8, 9, 10)", "ProductGroup", "https://schema.org/size"}})
}

func TestCheckFindsWhatAListingShows(t *testing.T) {
	r := checkShop(t, "qod-category", "")
	expectWarnings(t, r, [][]string{
		{"lists 24 products", "Tree of Life Heart Essential Oil Diffuser Locket Pendant Necklace", "ItemList", "CollectionPage"},
		{"1,153 pieces", "numberOfItems 1153"},
		{"FAQ section", "4 questions", "What length are the necklace chains?", "no FAQPage"},
		{"was prices", "$15.95"},
		{"Only 3 left", "inventoryLevel"},
		{"wishlist", "WantAction"},
	})
}

func TestCheckFindsAShopsPolicies(t *testing.T) {
	r := checkShop(t, "qod-shipping", "")
	expectWarnings(t, r, [][]string{
		{"$7.95", "ShippingService", "ShippingConditions"},
		{"free shipping over $60", "orderValue"},
		{"0-1 business days", "handlingTime"},
		{"2-6 business days", "transitTime"},
	})
	r = checkShop(t, "qod-returns", fixtureDraft(t, "qod-home"))
	expectWarnings(t, r, [][]string{{"merchantReturnLink", "http://localhost:8001/au/returns"}})
	r = checkShop(t, "qod-returns", "")
	expectWarnings(t, r, [][]string{{"within 14 days", "merchantReturnDays 14", "ReturnByMail", "ReturnFeesCustomerResponsibility"}})
	r = checkShop(t, "qod-contact", "")
	expectWarnings(t, r, [][]string{
		{"ContactPage"},
		{"ContactPoint", "customer service"},
		{"based in Australia", "PostalAddress", "addressCountry"},
	})
}

func TestCheckNamesTheStoreTypeForItsTrade(t *testing.T) {
	r := checkShop(t, "shop-store", "")
	expectWarnings(t, r, [][]string{
		{"street address", "JewelryStore"},
		{"opening hours", "OpeningHoursSpecification"},
		{"map", "GeoCoordinates"},
		{"newsletter form", "SubscribeAction"},
		{"Track your order", "TrackAction"},
	})
}

// A page whose markup carries everything the page shows passes: each shop
// check's demand is one a draft can meet.
func TestCheckPassesAShopPageMarkedUpInFull(t *testing.T) {
	for _, key := range []string{"qod-home", "qod-product-sale", "qod-product-variants", "qod-category", "qod-shipping", "qod-contact"} {
		t.Run(key, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("testdata", "shop", strings.TrimSuffix(shopPages[key].file, ".html")+"-full.jsonld"))
			if err != nil {
				t.Fatal(err)
			}
			r := checkShop(t, key, string(b))
			for _, i := range r.Issues {
				if i.Severity != Info {
					t.Errorf("%s %s: %s", i.Severity, i.Path, i.Message)
				}
			}
		})
	}
}
