package vocabtool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The ecommerce audit gives every type in the vocabulary a verdict for an
// online shop: the page whose build_jsonld report must name it, or why a
// shop does not use it. Every type must have one, and every type a shop uses
// must come up in the report for the page that calls for it, both with
// nothing marked up yet (an empty draft) and with the markup the page
// carries now. docs/ecommerce-schema-audit.md is written from the same
// tables: SCHEMA_MCP_WRITE_AUDIT=1 go test ./internal/vocabtool -run
// TestEcommerceAudit rewrites it.

// shopPage is a fixture: a rendered page under testdata/shop and the address
// it was served at. Its draft, when it has one, is the file of the same name
// ending .jsonld.
type shopPage struct {
	file string
	url  string
	what string
}

var shopPages = map[string]shopPage{
	"qod-home":             {"qod-home.html", "http://localhost:8001/au", "the Queen of Diamonds home page"},
	"qod-category":         {"qod-category.html", "http://localhost:8001/au/categories/necklace", "a Queen of Diamonds category page"},
	"qod-search":           {"qod-search.html", "http://localhost:8001/au/search?q=moon", "the Queen of Diamonds search results (noindex)"},
	"qod-product-variants": {"qod-product-variants.html", "http://localhost:8001/au/products/the-twilight-saga-bella-swan-vampire-crystal-ring", "a Queen of Diamonds product in five ring sizes"},
	"qod-product-sale":     {"qod-product-sale.html", "http://localhost:8001/au/products/tree-of-life-heart-essential-oil-diffuser-locket-pendant-necklace", "a Queen of Diamonds product on sale"},
	"qod-product-set":      {"qod-product-set.html", "http://localhost:8001/au/products/2-pcs-ruby-red-crystal-enchanted-vines-leaf-ring-set", "a Queen of Diamonds two-ring set with two left"},
	"qod-shipping":         {"qod-shipping.html", "http://localhost:8001/au/shipping", "the Queen of Diamonds shipping page"},
	"qod-returns":          {"qod-returns.html", "http://localhost:8001/au/returns", "the Queen of Diamonds return policy"},
	"qod-contact":          {"qod-contact.html", "http://localhost:8001/au/contact", "the Queen of Diamonds contact page"},
	"qod-terms":            {"qod-terms.html", "http://localhost:8001/au/terms", "the Queen of Diamonds terms page"},
	"qod-cart":             {"qod-cart.html", "http://localhost:8001/au/cart", "the Queen of Diamonds shopping bag"},
	"shop-product":         {"shop-product.html", "https://harbour.test/products/aurora-opal-pendant", "a generic shop's product page"},
	"shop-preorder":        {"shop-preorder.html", "https://harbour.test/products/art-deco-ring-set", "a generic shop's pre-order of a pre-owned set"},
	"shop-collection":      {"shop-collection.html", "https://harbour.test/collections/rings", "a generic shop's collection page"},
	"shop-store":           {"shop-store.html", "https://harbour.test/pages/contact", "a generic shop's store and contact page"},
	"shop-about":           {"shop-about.html", "https://harbour.test/pages/about", "a generic shop's about page"},
	"shop-checkout":        {"shop-checkout.html", "https://harbour.test/checkout", "a generic shop's checkout"},
	"content":              {"", "https://x.test/tea/", "the general content checks' article page (contentPage)"},
}

// shopUse is a type a shop's pages call for: the page whose report must name
// it, the words that name it when they are not its own name (an
// enumeration's member), and where it goes in the graph.
type shopUse struct {
	page   string
	expect string
	where  string
}

var shopUses = map[string]shopUse{
	// every page
	"WebSite":                    {"qod-product-sale", "", "the sitewide WebSite (url, name, inLanguage, publisher) on every page"},
	"WebPage":                    {"qod-terms", "", "every page's own node; policy and legal pages stay a plain WebPage"},
	"Organization":               {"qod-terms", "", "the business, typed as the subtype that fits"},
	"OnlineBusiness":             {"qod-terms", "", "parent of OnlineStore, the type of a business that trades online"},
	"OnlineStore":                {"qod-terms", "", "the business that sells online: name, url, logo, sameAs, email, taxID, hasMerchantReturnPolicy, hasShippingService"},
	"ImageObject":                {"qod-terms", "", "the business's logo, and each content image"},
	"OfferCatalog":               {"qod-terms", "", "the business's hasOfferCatalog: the categories its menu lists"},
	"SearchAction":               {"qod-terms", "", "the WebSite's potentialAction, for the site's search page"},
	"FollowAction":               {"qod-terms", "", "each social profile the page links, beside the business's sameAs"},
	"SiteNavigationElement":      {"qod-terms", "", "each nav, in the WebPage's hasPart"},
	"WPHeader":                   {"qod-terms", "", "the site header, in the WebPage's hasPart"},
	"WPFooter":                   {"qod-terms", "", "the site footer, in the WebPage's hasPart"},
	"EntryPoint":                 {"qod-product-sale", "", "an action's target: urlTemplate, httpMethod, actionPlatform"},
	"DigitalPlatformEnumeration": {"qod-product-sale", "DesktopWebPlatform", "an EntryPoint's actionPlatform"},
	"BreadcrumbList":             {"qod-product-sale", "", "the breadcrumb nav, as the WebPage's breadcrumb"},
	"ListItem":                   {"qod-category", "", "each entry of an ItemList or BreadcrumbList"},
	// product pages
	"ItemPage":                      {"qod-product-sale", "", "a product page's WebPage"},
	"Product":                       {"qod-product-sale", "", "the product a page sells, the ItemPage's mainEntity"},
	"Brand":                         {"qod-product-sale", "", "the Product's brand"},
	"Offer":                         {"qod-product-sale", "", "the Product's offers: price, priceCurrency, availability, itemCondition, seller"},
	"UnitPriceSpecification":        {"qod-product-sale", "", "a sale's was price: priceType StrikethroughPrice"},
	"PriceTypeEnumeration":          {"qod-product-sale", "StrikethroughPrice", "a UnitPriceSpecification's priceType"},
	"PriceSpecification":            {"qod-product-sale", "", "valueAddedTaxIncluded, for prices shown with GST"},
	"OfferShippingDetails":          {"qod-product-sale", "", "the Offer's shippingDetails"},
	"ShippingDeliveryTime":          {"qod-product-sale", "", "an OfferShippingDetails's deliveryTime: handlingTime and transitTime"},
	"PropertyValue":                 {"qod-product-sale", "", "a product detail with no property of its own, in additionalProperty"},
	"BuyAction":                     {"qod-product-sale", "", "the add-to-bag form, as the Offer's potentialAction"},
	"QuantitativeValue":             {"qod-product-set", "", "the Offer's inventoryLevel, and times and amounts"},
	"ItemAvailability":              {"qod-product-set", "LimitedAvailability", "the Offer's availability"},
	"ProductCollection":             {"qod-product-set", "", "a set of pieces sold as one, with includesObject"},
	"TypeAndQuantityNode":           {"qod-product-set", "", "each piece of a ProductCollection, with amountOfThisGood"},
	"ProductGroup":                  {"qod-product-variants", "", "a product in sizes or colours: variesBy, hasVariant, productGroupID"},
	"SizeSpecification":             {"qod-product-variants", "", "a variant's size, with its measurement and size system"},
	"WearableSizeSystemEnumeration": {"qod-product-variants", "", "a SizeSpecification's sizeSystem"},
	"WearableSizeGroupEnumeration":  {"qod-product-variants", "", "a SizeSpecification's sizeGroup"},
	// listings
	"CollectionPage":    {"qod-category", "", "a category or collection page's WebPage"},
	"ItemList":          {"qod-category", "", "the products a listing shows, the CollectionPage's mainEntity"},
	"AggregateOffer":    {"qod-category", "", "a listing's price range (lowPrice, highPrice)"},
	"FAQPage":           {"qod-category", "", "a page's questions, as a part of the page"},
	"Question":          {"qod-category", "", "each question an FAQ asks"},
	"Answer":            {"qod-category", "", "each question's acceptedAnswer"},
	"WantAction":        {"qod-category", "", "the wishlist buttons, as each Product's potentialAction"},
	"SearchResultsPage": {"qod-search", "", "the search page's WebPage (it is noindex, so the report only notes it)"},
	// policies
	"ShippingService":             {"qod-shipping", "", "the business's hasShippingService"},
	"ShippingConditions":          {"qod-shipping", "", "a ShippingService's rates, free-shipping order value and transit time"},
	"ServicePeriod":               {"qod-shipping", "", "handlingTime and transitTime in business days"},
	"DefinedRegion":               {"qod-shipping", "", "shippingDestination"},
	"MonetaryAmount":              {"qod-shipping", "", "shippingRate and orderValue"},
	"ShippingRateSettings":        {"qod-shipping", "", "the other way to give a free-shipping threshold (freeShippingThreshold)"},
	"DeliveryChargeSpecification": {"qod-shipping", "", "a ShippingRateSettings's freeShippingThreshold"},
	"FulfillmentTypeEnumeration":  {"qod-shipping", "FulfillmentTypeDelivery", "a ShippingService's fulfillmentType"},
	"MerchantReturnPolicy":        {"qod-returns", "", "the business's (and each Offer's) hasMerchantReturnPolicy"},
	"MerchantReturnEnumeration":   {"qod-returns", "MerchantReturnFiniteReturnWindow", "a MerchantReturnPolicy's returnPolicyCategory"},
	"ReturnMethodEnumeration":     {"qod-returns", "ReturnByMail", "a MerchantReturnPolicy's returnMethod"},
	"ReturnFeesEnumeration":       {"qod-returns", "ReturnFeesCustomerResponsibility", "a MerchantReturnPolicy's returnFees"},
	"ContactPage":                 {"qod-contact", "", "the contact page's WebPage"},
	"ContactPoint":                {"qod-contact", "", "the business's contactPoint: contactType, email, telephone"},
	"PostalAddress":               {"qod-contact", "", "the business's address, at least its addressCountry"},
	// what QoD does not show, on a generic shop's pages
	"Country":                              {"shop-product", "", "a Product's countryOfOrigin"},
	"IndividualProduct":                    {"shop-product", "", "a one-of-a-kind piece"},
	"PeopleAudience":                       {"shop-product", "", "a Product's audience, for a product named for women, men or children"},
	"GenderType":                           {"shop-product", "Female", "a PeopleAudience's suggestedGender"},
	"AggregateRating":                      {"shop-product", "", "the Product's aggregateRating, for the stars and review count it shows"},
	"Rating":                               {"shop-product", "", "each Review's reviewRating"},
	"Review":                               {"shop-product", "", "each customer review the page quotes"},
	"VideoObject":                          {"shop-product", "", "a product video"},
	"3DModel":                              {"shop-product", "", "a product's 3D view"},
	"Certification":                        {"shop-product", "", "a certificate the product carries (hasCertification)"},
	"WarrantyPromise":                      {"shop-product", "", "the Offer's warranty"},
	"MemberProgram":                        {"shop-product", "", "the business's hasMemberProgram, for member prices and points"},
	"MemberProgramTier":                    {"shop-product", "", "the tier a member price is validForMemberTier"},
	"TierBenefitEnumeration":               {"shop-product", "TierBenefitLoyaltyPrice", "a MemberProgramTier's hasTierBenefit"},
	"PaymentMethod":                        {"shop-product", "", "the business's acceptedPaymentMethod"},
	"PaymentCard":                          {"shop-product", "", "a card brand among the acceptedPaymentMethod"},
	"SaleEvent":                            {"shop-product", "", "a dated sale"},
	"ShareAction":                          {"shop-product", "", "the share buttons"},
	"ReviewAction":                         {"shop-product", "", "a write-a-review link"},
	"Table":                                {"shop-product", "", "a specification table, in the WebPage's hasPart"},
	"OfferItemCondition":                   {"shop-preorder", "UsedCondition", "the Offer's itemCondition"},
	"PreOrderAction":                       {"shop-preorder", "", "a pre-order button"},
	"PaymentChargeSpecification":           {"shop-preorder", "", "a card surcharge"},
	"ItemListOrderType":                    {"shop-collection", "ItemListOrderAscending", "an ItemList's itemListOrder"},
	"WPSideBar":                            {"shop-collection", "", "a filter sidebar, in the WebPage's hasPart"},
	"LocalBusiness":                        {"shop-store", "", "a business with a shopfront: the Store subtype for its trade"},
	"Store":                                {"shop-store", "", "parent of the shop types for each trade"},
	"OpeningHoursSpecification":            {"shop-store", "", "the store's openingHoursSpecification"},
	"DayOfWeek":                            {"shop-store", "", "an OpeningHoursSpecification's dayOfWeek"},
	"GeoCoordinates":                       {"shop-store", "", "the store's geo, beside hasMap"},
	"SubscribeAction":                      {"shop-store", "", "a newsletter form"},
	"TrackAction":                          {"shop-store", "", "an order-tracking link"},
	"AskAction":                            {"shop-store", "", "a contact form"},
	"QuoteAction":                          {"shop-store", "", "a quote or custom-order form (named by the form check)"},
	"ServiceChannel":                       {"shop-store", "", "a reply-time promise, as processingTime"},
	"MerchantReturnPolicySeasonalOverride": {"shop-store", "", "holiday returns, as a returnPolicySeasonalOverride"},
	"RefundTypeEnumeration":                {"shop-store", "StoreCreditRefund", "a MerchantReturnPolicy's refundType"},
	"ReturnLabelSourceEnumeration":         {"shop-store", "ReturnLabelInBox", "a MerchantReturnPolicy's returnLabelSource"},
	"PaymentMethodType":                    {"shop-store", "ByBankTransferInAdvance", "a PaymentMethod's paymentMethodType"},
	"AboutPage":                            {"shop-about", "", "the about page's WebPage"},
	"Person":                               {"shop-about", "", "the founder, and each review's author"},
	"CheckoutPage":                         {"shop-checkout", "", "the checkout's WebPage (a private page, so the report only notes it)"},
	// the general content checks, for a shop's guides and blog
	"Article":        {"content", "", "a guide or blog post"},
	"HowTo":          {"content", "", "a care guide's steps"},
	"Quotation":      {"content", "", "a quotation that is not a review"},
	"DefinedTerm":    {"content", "", "a glossary term"},
	"DefinedTermSet": {"content", "", "a glossary"},
}

// storeNames are businesses with a shopfront, one for each Store subtype:
// their name says their trade, and the report must name the subtype.
var storeNames = map[string]string{
	"AutoPartsStore":       "Westside Auto Parts",
	"BikeStore":            "Spoke and Chain Bicycles",
	"BookStore":            "Riverside Books",
	"ClothingStore":        "Harbour Fashion Boutique",
	"ComputerStore":        "Byte Computers",
	"ConvenienceStore":     "Corner Convenience",
	"DepartmentStore":      "Grand Department Store",
	"ElectronicsStore":     "Volt Electronics",
	"Florist":              "Bloom Florist",
	"FurnitureStore":       "Oak Furniture",
	"GardenStore":          "Greenleaf Garden Centre",
	"GroceryStore":         "Fresh Grocer",
	"HardwareStore":        "Bolt Hardware",
	"HobbyShop":            "Craft and Hobby Shop",
	"HomeGoodsStore":       "Nest Homewares",
	"JewelryStore":         "Harbour Jewellers",
	"LiquorStore":          "Cellar Door Liquor",
	"MensClothingStore":    "Tailor Menswear",
	"MobilePhoneStore":     "Signal Mobile Phones",
	"MovieRentalStore":     "Reel Movie Rental",
	"MusicStore":           "Chord Music Store",
	"OfficeEquipmentStore": "Desk Office Supplies",
	"OutletStore":          "Brand Outlet",
	"PawnShop":             "Fair Deal Pawnbrokers",
	"PetStore":             "Paws Pet Supplies",
	"ShoeStore":            "Stride Shoes",
	"SportingGoodsStore":   "Field Sports",
	"TireShop":             "Grip Tyres",
	"ToyStore":             "Toy Box",
	"WholesaleStore":       "Bulk Wholesale",
}

// notForShop are types no rule in auditReason excludes that a shop still
// does not use, or that a rule would describe wrongly.
var notForShop = map[string]string{
	"Thing":                           "the root type: every node takes a more specific one",
	"CreativeWork":                    "a parent type: the report names the subtype that fits",
	"Intangible":                      "a parent type: the report names the subtype that fits",
	"StructuredValue":                 "a parent type: the report names the subtype that fits",
	"Action":                          "a parent type: the report names the subtype that fits",
	"Event":                           "a parent type: the report names the subtype that fits",
	"Place":                           "a parent type: the report names the subtype that fits",
	"MediaObject":                     "a parent type: the report names the subtype that fits",
	"WebPageElement":                  "a parent type: the report names the subtype that fits",
	"Enumeration":                     "a parent type: the report names the subtype that fits",
	"QualitativeValue":                "a parent type: the report names the subtype that fits",
	"Audience":                        "a parent type: the report names the subtype that fits",
	"CommunicateAction":               "a parent type: the report names the subtype that fits",
	"SizeSystemEnumeration":           "a parent type: the report names WearableSizeSystemEnumeration",
	"SizeGroupEnumeration":            "a parent type: the report names WearableSizeGroupEnumeration",
	"SomeProducts":                    "a quantity of unnamed products: each catalogue item is a Product",
	"ProductModel":                    "a manufacturer's model of a product line: a retailer lists the Product",
	"Demand":                          "a wish to buy, not an offer to sell",
	"OfferForPurchase":                "an Offer already says it is for sale",
	"OfferForLease":                   "a lease, not a sale",
	"UserReview":                      "a customer's review is a Review",
	"CriticReview":                    "a customer's review is a Review",
	"EmployerReview":                  "a customer's review is a Review",
	"MediaReview":                     "a customer's review is a Review",
	"ClaimReview":                     "a customer's review is a Review",
	"Recommendation":                  "a customer's review is a Review",
	"EmployerAggregateRating":         "rates an employer, not a product",
	"EndorsementRating":               "an endorsement count, not a product rating",
	"CreditCard":                      "a card brand is a PaymentCard",
	"LoanOrCredit":                    "a financial product: a pay-later service is named as a PaymentMethod",
	"FinancialProduct":                "a financial product: a pay-later service is named as a PaymentMethod",
	"Order":                           "order details live in emails and signed-in pages, not public pages",
	"OrderItem":                       "order details live in emails and signed-in pages, not public pages",
	"ParcelDelivery":                  "order details live in emails and signed-in pages, not public pages",
	"Invoice":                         "order details live in emails and signed-in pages, not public pages",
	"ProgramMembership":               "a customer's membership lives in their signed-in account",
	"CompoundPriceSpecification":      "a price built of parts (a deposit and instalments): a shop shows one price per item",
	"ImageGallery":                    "a gallery page: a product's photos are its image list",
	"MediaGallery":                    "a gallery page: a product's photos are its image list",
	"VideoGallery":                    "a gallery page: a product's videos are VideoObjects",
	"Language":                        "inLanguage takes the page's language code as text",
	"Service":                         "a business that sells services: a shop sells products",
	"BusinessAudience":                "the shop sells to people: PeopleAudience",
	"ParentAudience":                  "the shop sells to people: PeopleAudience",
	"EducationalAudience":             "the shop sells to people: PeopleAudience",
	"MedicalAudience":                 "the shop sells to people: PeopleAudience",
	"DataFeed":                        "a product feed goes to Merchant Center, not into a page",
	"DataFeedItem":                    "a product feed goes to Merchant Center, not into a page",
	"CompleteDataFeed":                "a product feed goes to Merchant Center, not into a page",
	"CategoryCode":                    "a category code (such as Google's product taxonomy) is set by choice; no page shows one",
	"CategoryCodeSet":                 "a category code (such as Google's product taxonomy) is set by choice; no page shows one",
	"WPAdBlock":                       "an ad slot: the page-sections check names it on pages that run ads",
	"MediaReviewItem":                 "a fact-check item, not a shop concept",
	"BlogPosting":                     "QoD has no blog; the general checks name a post an Article",
	"EnergyConsumptionDetails":        "an energy label, for an appliance shop: not detected",
	"Vehicle":                         "sold by dealers, not from a shop catalogue",
	"Car":                             "sold by dealers, not from a shop catalogue",
	"Motorcycle":                      "sold by dealers, not from a shop catalogue",
	"BusOrCoach":                      "sold by dealers, not from a shop catalogue",
	"MotorizedBicycle":                "sold by dealers, not from a shop catalogue",
	"WebContent":                      "a block of web content: the page's own type says more",
	"DigitalProductPassport":          "a product passport document, published by the maker",
	"EnvironmentalProductDeclaration": "a maker's environmental declaration document",
	"DeclarationOfConformity":         "a maker's conformity document",
	"PronounceableText":               "text with its pronunciation, for speech: not a shop concept",
}

// auditReason is why a type no shop page calls for is left out, from the
// branch it sits in.
func auditReason(t *Tool, name string) string {
	c := t.v.Classes[name]
	anc := t.v.Ancestors(name)
	under := func(ty string) bool { return slices.Contains(anc, ty) && name != ty }
	switch {
	case c.DataType || !t.v.IsStruct(name):
		return "a value written as text, a number or a date, not a node"
	case len(c.SupersededBy) > 0:
		return "superseded: the report names its replacement"
	case c.Part == "attic":
		return "retired to the attic"
	case c.Part == "meta":
		return "a term about the vocabulary itself"
	case c.Part == "health-lifesci" || under("MedicalEntity") || under("BioChemEntity") || under("Taxon") || name == "MedicalEntity" || name == "BioChemEntity" || name == "Taxon":
		return "a medical or life-science type"
	case under("Store"):
		return ""
	case under("LocalBusiness"):
		return "a local business of another trade"
	case under("Organization"):
		return "an organization that is not a shop"
	case under("Enumeration"):
		return "an enumeration no property a shop page uses takes"
	case under("Action"):
		return "an action a shop page does not offer"
	case under("Event"):
		return "an event: a shop page holds none but a sale (SaleEvent)"
	case under("Place"):
		return "a place a shop page does not describe"
	case under("CreativeWork"):
		return "a creative work a shop does not publish"
	case under("Product"):
		return "a product kind a shop catalogue does not hold"
	case under("Intangible"):
		return "an intangible a shop page does not describe"
	}
	return ""
}

// auditVerdict is a type's use on a shop's pages, or why it has none. A
// type with neither has no verdict yet.
func auditVerdict(t *Tool, name string) (*shopUse, string) {
	if u, ok := shopUses[name]; ok {
		return &u, ""
	}
	if _, ok := storeNames[name]; ok {
		return &shopUse{page: "store:" + name, where: "a business with a shopfront whose trade is " + name}, ""
	}
	if r, ok := notForShop[name]; ok {
		return nil, r
	}
	return nil, auditReason(t, name)
}

// shopReport is the union of the reports for a fixture: with an empty draft,
// and with the draft the page carries now when it has one.
func shopReport(t *testing.T, tool *Tool, key string) []Issue {
	t.Helper()
	var page, base string
	var drafts []string
	if name, ok := strings.CutPrefix(key, "store:"); ok {
		biz := storeNames[name]
		page = `<!doctype html><html lang="en"><head><title>Visit ` + biz + `</title><link rel="canonical" href="https://store.test/visit"></head><body>
<main><h1>Visit ` + biz + `</h1><address>1 Main Street, Springfield</address><p>Open Monday to Saturday 9:00am to 5:00pm.</p></main>
<footer><p>© 2026 ` + biz + `</p></footer></body></html>`
		base = "https://store.test/visit"
	} else {
		p, ok := shopPages[key]
		if !ok {
			t.Fatalf("no fixture %q", key)
		}
		base = p.url
		page = contentPage
		if p.file != "" {
			b, err := os.ReadFile(filepath.Join("testdata", "shop", p.file))
			if err != nil {
				t.Fatal(err)
			}
			page = string(b)
			if d, err := os.ReadFile(filepath.Join("testdata", "shop", strings.TrimSuffix(p.file, ".html")+".jsonld")); err == nil {
				drafts = append(drafts, string(d))
			}
		}
	}
	var out []Issue
	for _, d := range append([]string{`{"@context":"https://schema.org","@graph":[]}`}, drafts...) {
		r, err := tool.Check(json.RawMessage(d), CheckOptions{PageHTML: page, PageURL: base})
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		out = append(out, r.Issues...)
	}
	return out
}

// names reports whether an issue names the words as a whole term.
func reportNames(issues []Issue, words string) bool {
	re := regexp.MustCompile(`(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(words) + `(?:$|[^A-Za-z0-9])`)
	return slices.ContainsFunc(issues, func(i Issue) bool { return re.MatchString(i.Message) })
}

func TestEcommerceAudit(t *testing.T) {
	tool := newTool(t)
	names := make([]string, 0, len(tool.v.Classes))
	for name := range tool.v.Classes {
		names = append(names, name)
	}
	sort.Strings(names)
	reports := map[string][]Issue{}
	var used []string
	skipped := map[string][]string{}
	var unjudged []string
	for _, name := range names {
		use, reason := auditVerdict(tool, name)
		switch {
		case use != nil:
			used = append(used, name)
			if _, ok := reports[use.page]; !ok {
				reports[use.page] = shopReport(t, tool, use.page)
			}
			words := use.expect
			if words == "" {
				words = name
			}
			if !reportNames(reports[use.page], words) {
				t.Errorf("%s: the report for %s does not name %s", name, use.page, words)
			}
		case reason != "":
			skipped[reason] = append(skipped[reason], name)
		default:
			unjudged = append(unjudged, name)
		}
	}
	if len(unjudged) > 0 {
		t.Errorf("%d types have no verdict: %s", len(unjudged), strings.Join(unjudged, ", "))
	}
	for name := range shopUses {
		if _, ok := tool.v.Classes[name]; !ok {
			t.Errorf("shopUses names %s, which the vocabulary lacks", name)
		}
	}
	doc := auditDoc(tool, names, used, skipped)
	path := filepath.Join("..", "..", "docs", "ecommerce-schema-audit.md")
	if os.Getenv("SCHEMA_MCP_WRITE_AUDIT") == "1" {
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if old, err := os.ReadFile(path); err != nil || string(old) != doc {
		t.Errorf("%s is stale: run SCHEMA_MCP_WRITE_AUDIT=1 go test ./internal/vocabtool -run TestEcommerceAudit", path)
	}
}

// auditDoc is the audit as markdown.
func auditDoc(tool *Tool, names, used []string, skipped map[string][]string) string {
	var b strings.Builder
	b.WriteString("# Ecommerce schema audit\n\n")
	b.WriteString("<!-- Written by TestEcommerceAudit (internal/vocabtool/audit_test.go). Do not edit: change the tables there and run SCHEMA_MCP_WRITE_AUDIT=1 go test ./internal/vocabtool -run TestEcommerceAudit. -->\n\n")
	fmt.Fprintf(&b, "Every one of the vocabulary's %d types, judged for an online shop such as Queen of Diamonds Jewellery. ", len(names))
	fmt.Fprintf(&b, "A shop uses %d of them. For each, `build_jsonld` (given the rendered page as `page_html` and its address as `page_url`) names the type on the page listed, ", len(used))
	b.WriteString("both with nothing marked up and with the markup the page carries now, and the test fails if it stops doing so. ")
	fmt.Fprintf(&b, "The other %d are left out, for the reason given.\n\n", len(names)-len(used))
	b.WriteString("## Pages\n\n| Page | What it is |\n|---|---|\n")
	keys := make([]string, 0, len(shopPages))
	for k := range shopPages {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := shopPages[k]
		file := "`contentPage` in content_test.go"
		if p.file != "" {
			file = "`testdata/shop/" + p.file + "`"
		}
		fmt.Fprintf(&b, "| %s (%s) | %s |\n", k, file, p.what)
	}
	b.WriteString("| store:*Type* | a page built in the test for each Store subtype: a business whose name says its trade, with a street address |\n")
	b.WriteString("\n## Types a shop uses\n\n| Type | Where it goes | Named on |\n|---|---|---|\n")
	for _, name := range used {
		u, _ := auditVerdict(tool, name)
		on := u.page
		if u.expect != "" {
			on += " (as " + u.expect + ")"
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", name, u.where, on)
	}
	b.WriteString("\n## Types a shop does not use\n")
	reasons := make([]string, 0, len(skipped))
	for r := range skipped {
		reasons = append(reasons, r)
	}
	sort.Slice(reasons, func(i, j int) bool {
		if len(skipped[reasons[i]]) != len(skipped[reasons[j]]) {
			return len(skipped[reasons[i]]) > len(skipped[reasons[j]])
		}
		return reasons[i] < reasons[j]
	})
	for _, r := range reasons {
		fmt.Fprintf(&b, "\n**%s** (%d): %s\n", strings.ToUpper(r[:1])+r[1:], len(skipped[r]), strings.Join(skipped[r], ", "))
	}
	return b.String()
}
