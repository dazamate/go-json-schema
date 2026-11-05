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

// trades name the schema.org type a business with premises takes from the
// words its name, title and heading use, most specific first. A retail trade
// takes a Store subtype and a service trade the LocalBusiness subtype for it.
// A word that names a trade also names a thing a service business mentions
// ("books" in "Book now", "pets" in "pet friendly", "garden" in lawn care), so
// each pattern names the trade's own kind of business: a shop, a store, a
// practice, a salon. An entry with no type is a trade that is no shop and has
// no subtype: it stops a later entry from reading it as one.
var trades = []struct {
	words *regexp.Regexp
	use   string
}{
	// a trade that is a shop although its words also name a service
	{regexp.MustCompile(`(?i)\b(garden|plant) nurser(y|ies)\b|\bgarden (centres?|centers?|stores?|shops?|supplies)\b`), "GardenStore"},
	{regexp.MustCompile(`(?i)\b(pet|animal) (shops?|stores?|supplies|food|warehouse|centres?|centers?)\b|\bpet ?(barn|stop|land|world)\b`), "PetStore"},
	{regexp.MustCompile(`(?i)\b(yoga|gym|fitness|activewear|workout) (mats?|pants|wear|clothing|leggings|gear|equipment)\b|\b(sports?|sporting) (goods|stores?|shops?|gear|equipment|warehouse|apparel|wear)\b`), "SportingGoodsStore"},
	{regexp.MustCompile(`(?i)\belectrical (goods|appliances|supplies|stores?)\b`), "ElectronicsStore"},
	// a trade that is no shop
	{regexp.MustCompile(`(?i)\b(furniture|piano|home|house|office) (removals?|moving|movers|relocation)\b|\bmovers\b|\bremovalists?\b|\bremovals\b|\bmoving (company|companies|services?)\b`), "MovingCompany"},
	{regexp.MustCompile(`(?i)\bself[- ]?storage\b|\bstorage units?\b`), "SelfStorage"},
	{regexp.MustCompile(`(?i)\bfurniture (assembly|repair|restoration|upholstery|hire|rental|delivery)\b`), ""},
	{regexp.MustCompile(`(?i)\b(dent(al|ists?|istry)|orthodont\w*)\b`), "Dentist"},
	{regexp.MustCompile(`(?i)\b(opticians?|optometr\w*|eye ?care)\b`), "Optician"},
	{regexp.MustCompile(`(?i)\b(veterinary|veterinarians?|vets?|animal (hospital|clinic))\b`), "VeterinaryCare"},
	{regexp.MustCompile(`(?i)\b(physio(therapy|therapists?)?|chiropract\w*|osteopath\w*)\b`), "MedicalClinic"},
	{regexp.MustCompile(`(?i)\b(pharmacy|pharmacies|chemists?)\b`), "Pharmacy"},
	{regexp.MustCompile(`(?i)\b(hair (salons?|studio|design|lounge|stylists?|dressers?|and beauty|&(amp;)? beauty)|hairdress\w*|barbers?(hop)?)\b`), "HairSalon"},
	{regexp.MustCompile(`(?i)\b(nail (salons?|bar|studio|spa|techs?|lounge|art)|nails)\b`), "NailSalon"},
	{regexp.MustCompile(`(?i)\b(day spas?|spas?|massage|wellness (centre|center))\b`), "DaySpa"},
	{regexp.MustCompile(`(?i)\b(beauty (salons?|clinic|studio|therap\w+|rooms?|parlou?r|lounge|bar)|beauticians?|salons?|lash\w*|waxing|aesthetics?|tanning)\b`), "BeautySalon"},
	{regexp.MustCompile(`(?i)\btattoo\w*`), "TattooParlor"},
	{regexp.MustCompile(`(?i)\b(medical (centre|center|clinic|practice)|health (centre|center)|clinics?|general practice|family practice|urgent care|surgery)\b`), "MedicalClinic"},
	{regexp.MustCompile(`(?i)\b(doctors?|physicians?|general practitioners?|gp (clinic|surgery))\b`), "Physician"},
	{regexp.MustCompile(`(?i)\bnotar(y|ies)\b`), "Notary"},
	{regexp.MustCompile(`(?i)\battorneys?\b`), "Attorney"},
	{regexp.MustCompile(`(?i)\b(lawyers?|solicitors?|barristers?|law|legal (services?|practice|advice|centre|center)|conveyancing)\b`), "LegalService"},
	{regexp.MustCompile(`(?i)\b(accountants?|accounting|accountancy|bookkeep\w*|tax (agents?|returns?|preparation|services?)|cpas?)\b`), "AccountingService"},
	{regexp.MustCompile(`(?i)\binsurance\b`), "InsuranceAgency"},
	{regexp.MustCompile(`(?i)\b(building society|credit union|banking)\b`), "BankOrCreditUnion"},
	{regexp.MustCompile(`(?i)\b(mortgages?|financial (planning|planners?|advis[eo]rs?|services?)|wealth management)\b`), "FinancialService"},
	{regexp.MustCompile(`(?i)\b(real estate|realty|realtors?|estate agents?|letting agents?|property (management|managers?|agents?))\b`), "RealEstateAgent"},
	{regexp.MustCompile(`(?i)plumb\w*|\b(drain\w*|gas ?fitt\w*)\b`), "Plumber"},
	{regexp.MustCompile(`(?i)electricians?|\belectrical\b`), "Electrician"},
	{regexp.MustCompile(`(?i)\b(hvac|air[- ]?con\w*|heating|cooling|furnaces?|heat pumps?|boilers?)\b`), "HVACBusiness"},
	{regexp.MustCompile(`(?i)\broof(ers?|ing)\b`), "RoofingContractor"},
	{regexp.MustCompile(`(?i)locksmiths?`), "Locksmith"},
	{regexp.MustCompile(`(?i)\b(painters?|decorators?|house painting)\b`), "HousePainter"},
	{regexp.MustCompile(`(?i)\b(builders?|general contractors?|building contractors?|construction (company|services?)|remodel\w*|renovations?|handyman)\b`), "GeneralContractor"},
	{regexp.MustCompile(`(?i)\bmotels?\b`), "Motel"},
	{regexp.MustCompile(`(?i)\bbed (and|&(amp;)?|n) breakfast\b|\bb ?&(amp;)? ?b\b`), "BedAndBreakfast"},
	{regexp.MustCompile(`(?i)\bhostels?\b`), "Hostel"},
	{regexp.MustCompile(`(?i)\b(camp ?grounds?|caravan parks?|holiday parks?)\b`), "Campground"},
	{regexp.MustCompile(`(?i)\bresorts?\b`), "Resort"},
	{regexp.MustCompile(`(?i)\b(hotels?|inns?|guest ?houses?|serviced apartments?|accommodation)\b`), "Hotel"},
	{regexp.MustCompile(`(?i)\b(bakery|bakeries|bakehouse|bakers?|patisserie)\b`), "Bakery"},
	{regexp.MustCompile(`(?i)\bcafes?\b|\bcafé|\b(coffee (shop|house|bar|roasters?))\b`), "CafeOrCoffeeShop"},
	{regexp.MustCompile(`(?i)\b(pubs?|taverns?|taproom|gastropub|(wine|cocktail|sports|tapas|beer|whisky|piano) bar|bar (and|&(amp;)?) grill)\b`), "BarOrPub"},
	{regexp.MustCompile(`(?i)\b(breweries|brewery|brewing co)\b`), "Brewery"},
	{regexp.MustCompile(`(?i)\b(wineries|winery|vineyard)\b`), "Winery"},
	{regexp.MustCompile(`(?i)\b(ice ?cream|gelato)\b`), "IceCreamShop"},
	{regexp.MustCompile(`(?i)\b(restaurants?|bistro|brasserie|trattoria|eatery|diner|pizz(a|eria)|steak ?house|sushi|grill|ramen|tapas|grill ?house)\b`), "Restaurant"},
	{regexp.MustCompile(`(?i)\b(catering|food trucks?)\b`), "FoodEstablishment"},
	{regexp.MustCompile(`(?i)\b(health clubs?|yoga|pilates)\b`), "HealthClub"},
	{regexp.MustCompile(`(?i)\b(gyms?|fitness|crossfit|personal train\w*)\b`), "ExerciseGym"},
	{regexp.MustCompile(`(?i)\bcar wash\b`), "AutoWash"},
	{regexp.MustCompile(`(?i)\b((car|van|vehicle|truck) (rental|hire)s?|rent[- ]a[- ]car)\b`), "AutoRental"},
	{regexp.MustCompile(`(?i)\b((auto|car|motor|vehicle|truck|diesel) (repairs?|service|servicing|mechanics?|workshop|care|body|collision|detailing)|mechanics?|panel ?beaters?|smash repairs?|automotive)\b`), "AutoRepair"},
	{regexp.MustCompile(`(?i)\b(car|auto|motor|vehicle|truck) (dealers?|dealerships?|sales|yard|group|centre|center|mart)\b|\b(used|new|second[- ]hand) cars?\b|\bdealerships?\b`), "AutoDealer"},
	{regexp.MustCompile(`(?i)\b(child ?care|day ?care|nurser(y|ies)|pre-?school|kindergarten|early (learning|education)|cr[eè]che|playgroup|after[- ]school)\b`), "ChildCare"},
	{regexp.MustCompile(`(?i)\b(dry ?clean\w*|laundr\w+)\b`), "DryCleaningOrLaundry"},
	{regexp.MustCompile(`(?i)\btravel (agency|agents?|centre|center)\b`), "TravelAgency"},
	{regexp.MustCompile(`(?i)\b(recruit\w*|employment agency|staffing)\b`), "EmploymentAgency"},
	{regexp.MustCompile(`(?i)\bshopping (centre|center|mall)\b`), "ShoppingCenter"},
	// a retail trade
	{regexp.MustCompile(`(?i)\b(menswear|men'?s (clothing|wear|fashion))\b`), "MensClothingStore"},
	{regexp.MustCompile(`(?i)\b(jewell?e?ry|jewell?ers?)\b`), "JewelryStore"},
	{regexp.MustCompile(`(?i)\b(auto|car|motor|truck) parts\b|\bparts (stores?|shops?)\b`), "AutoPartsStore"},
	{regexp.MustCompile(`(?i)\b(bike|bicycle|cycle)s? (shops?|stores?|warehouse|centres?|centers?)\b`), "BikeStore"},
	{regexp.MustCompile(`(?i)\b(books|bookshop|bookstore|booksellers?|book (shops?|stores?))\b`), "BookStore"},
	{regexp.MustCompile(`(?i)\b(computer|laptop|pc) (stores?|shops?|warehouse|centres?|centers?|world)\b|\bcomputers? (and|&(amp;)?) (laptops?|tablets?|accessories)\b`), "ComputerStore"},
	{regexp.MustCompile(`(?i)\bconvenience (stores?|shops?)\b`), "ConvenienceStore"},
	{regexp.MustCompile(`(?i)\bdepartment stores?\b`), "DepartmentStore"},
	{regexp.MustCompile(`(?i)\b(electronics? (stores?|shops?|warehouse|retailers?)|consumer electronics)\b`), "ElectronicsStore"},
	{regexp.MustCompile(`(?i)\b(florists?|flowers)\b`), "Florist"},
	{regexp.MustCompile(`(?i)\bfurniture\b`), "FurnitureStore"},
	{regexp.MustCompile(`(?i)\b(grocery|grocers?|supermarkets?)\b`), "GroceryStore"},
	{regexp.MustCompile(`(?i)\bhardware (stores?|shops?|supplies|warehouse|centres?|centers?)\b|\b(home|trade) hardware\b`), "HardwareStore"},
	{regexp.MustCompile(`(?i)\b(hobby (shops?|stores?)|hobbies|(craft|art) (stores?|shops?|supplies))\b`), "HobbyShop"},
	{regexp.MustCompile(`(?i)\b(homewares?|home goods)\b`), "HomeGoodsStore"},
	{regexp.MustCompile(`(?i)\b(liquor|bottle shop|wine shop)\b`), "LiquorStore"},
	{regexp.MustCompile(`(?i)\b((mobile )?phones? (stores?|shops?|warehouse)|mobile phones? (and|&(amp;)?) (accessories|tablets))\b`), "MobilePhoneStore"},
	{regexp.MustCompile(`(?i)\b(movie|video) rentals?\b`), "MovieRentalStore"},
	{regexp.MustCompile(`(?i)\b(music (stores?|shops?|centres?|centers?|warehouse)|musical instruments?|record (stores?|shops?)|vinyl records)\b`), "MusicStore"},
	{regexp.MustCompile(`(?i)\boffice (supplies|equipment)\b|\bstationery\b`), "OfficeEquipmentStore"},
	{regexp.MustCompile(`(?i)\b(factory|clearance) outlet\b|\boutlet (stores?|shops?|centres?|centers?)\b`), "OutletStore"},
	{regexp.MustCompile(`(?i)\bpawn(brokers?|shop)?\b`), "PawnShop"},
	{regexp.MustCompile(`(?i)\b(shoes|footwear)\b`), "ShoeStore"},
	{regexp.MustCompile(`(?i)\b(tyres?|tires?)\b`), "TireShop"},
	{regexp.MustCompile(`(?i)\b(toys|toy (stores?|shops?))\b`), "ToyStore"},
	{regexp.MustCompile(`(?i)\bwholesale\b`), "WholesaleStore"},
	{regexp.MustCompile(`(?i)\b(clothing|fashion|apparel|boutique)\b`), "ClothingStore"},
}

// looseTrades name a shop by the goods alone, as a business name does
// ("Spoke and Chain Bicycles", "Volt Electronics"): they apply only when
// nothing names the business as a service, as notShop does.
var looseTrades = []struct {
	words *regexp.Regexp
	use   string
}{
	{regexp.MustCompile(`(?i)\b(bikes?|bicycles?)\b`), "BikeStore"},
	{regexp.MustCompile(`(?i)\b(computers?|laptops?)\b`), "ComputerStore"},
	{regexp.MustCompile(`(?i)\bconvenience\b`), "ConvenienceStore"},
	{regexp.MustCompile(`(?i)\belectronics\b`), "ElectronicsStore"},
	{regexp.MustCompile(`(?i)\bhardware\b`), "HardwareStore"},
	{regexp.MustCompile(`(?i)\b(mobile phones?|phones)\b`), "MobilePhoneStore"},
	{regexp.MustCompile(`(?i)\boutlet\b`), "OutletStore"},
	{regexp.MustCompile(`(?i)\b(sports?|sporting goods)\b`), "SportingGoodsStore"},
	{regexp.MustCompile(`(?i)\btoys?\b`), "ToyStore"},
}

// notShop is a word that makes a business a service whatever goods its name
// mentions: a repair, a lesson, a hire, a club, a call to book.
var notShop = regexp.MustCompile(`(?i)\b(repairs?|servic\w+|lessons?|hire|rentals?|tours?|classes|training|coaching|clinic|school|academy|club|centre|center|medicine|therapy|book (now|online|an?|your|today|with))\b`)

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

// localType is the type for a business whose page shows a street address: the
// type its trade takes, or Store when the page sells and LocalBusiness when it
// does not.
func localType(p *webpage.Facts) string {
	name := businessName(p) + " · " + p.H1()
	named := false
	for _, k := range trades {
		if k.words.MatchString(name) {
			if k.use != "" {
				return k.use
			}
			named = true
			break
		}
	}
	if !named && !notShop.MatchString(name) {
		for _, k := range looseTrades {
			if k.words.MatchString(name) {
				return k.use
			}
		}
	}
	if p.Sells() {
		return "Store"
	}
	return "LocalBusiness"
}

// shopBusiness is the business a shop's page calls for and the draft's node
// for it.
type shopBusiness struct {
	want string      // the type the business takes: the type for its trade, Store or LocalBusiness, or OnlineStore
	why  string      // what on the page calls for that type
	node jsonld.Node // the draft's first Organization with properties; N is nil when it has none
}

// shopBusinessKey is the memo key of the shop's business.
type shopBusinessKey struct{}

// shopBusinessOf is the business the page calls for, worked out once: a shop
// that sells online is an OnlineStore, and a business whose page shows a
// street address the type for its trade (a Store subtype for a shop, the
// LocalBusiness subtype of a service), even when it also sells online.
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
			b.want = localType(c.Page)
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
