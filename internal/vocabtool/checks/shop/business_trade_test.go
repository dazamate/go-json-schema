package shop_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
)

// tradePage is a business's contact page: the site name, title and heading
// are the real ones of a page of the trade, and the page offers booking and
// shows the street address. Booking and pets are everyday words in a service
// business's copy.
func tradePage(siteName, title, h1, address string) string {
	return fmt.Sprintf(`<!doctype html><html lang="en"><head><meta property="og:site_name" content="%s"><title>%s</title></head><body>
<header><a href="/">%s</a><nav><a href="/book">Book now</a> <a href="/pets">Pets welcome</a></nav></header>
<main><h1>%s</h1><p>Book an appointment today.</p><address>%s</address></main></body></html>`, siteName, title, siteName, h1, address)
}

func localStoreFor(t string, address string) []audit.Issue {
	a := "a"
	if strings.Contains("AEIOU", t[:1]) {
		a = "an"
	}
	return []audit.Issue{{Path: "html:body", Severity: audit.Warning, Message: fmt.Sprintf(`the page shows the street address %q but no business node: add the business as %s %s (the LocalBusiness subtype for its trade, a Store subtype for a shop: name, address, geo, openingHoursSpecification, telephone)`, address, a, t)}}
}

// A page with a street address belongs to a local business of its trade: a
// service business is no Store.
func TestLocalStoreServiceTrades(t *testing.T) {
	const url = "https://example.test/contact/"
	for _, tc := range []struct {
		name, site, title, h1, address, want string
	}{
		// dentistryforyou.co.uk
		{"dentist", "Dentistry for you", "Private treatment fees - Dentistry for you", "Private dental care treatment costs", "45 Southend Road, Hockley, SS5 4PZ", "Dentist"},
		// mooreheaddentistry.com
		{"dentist, US", "Moorehead Dentistry", "Dental Office Fees - Moorehead Dentistry - 3 Locations in Ohio", "Office Fees", "10427 Kenwood Road, Cincinnati, OH 45242", "Dentist"},
		// willowsbeautysalon.co.uk
		{"salon", "The Willows Beauty Salon", "Treatment Price List - The Willows Beauty Salon", "Treatment Price List", "8 Church Lane, Leeds LS1 4AB", "BeautySalon"},
		// opushotel.com
		{"hotel", "Opus Hotel Vancouver", "Vancouver Boutique Hotel in Yaletown", "Opus Hotel", "322 Davie St, Vancouver, BC V6B 5Z6", "Hotel"},
		// thegymgroup.com
		{"gym", "The Gym Group", "Gym Membership | The Gym Group", "Gym Membership", "10 Station Road, Leeds LS1 4AB", "ExerciseGym"},
		// smithpartners.co.nz
		{"law firm", "Smith & Partners", "Property Lawyers Auckland | Buying, Selling & Property Law", "Property Law", "293 Lincoln Road, Henderson, Auckland 0610", "LegalService"},
		// instaelectricians.com
		{"electrician", "InstaElectricians", "Electrical Panel Upgrade Cost (2026): Full Guide", "Electrical Panel Upgrade Cost: What Homeowners Should Expect", "20 Main Street, Austin, TX 78701", "Electrician"},
		// jamesplumbingheating.com
		{"plumber", "James Plumbing & Heating", "Pricing & Rates | James Plumbing & Heating", "Transparent Pricing", "15 Coniston Road, Twickenham TW2 7HH", "Plumber"},
		// vicisouthbank.com.au
		{"restaurant", "Pizza & Pasta Brisbane", "Enjoy Italian Food at Vici South Bank", "Our Menus", "174 Grey St, South Brisbane 4101", "Restaurant"},
		{"vet who welcomes pets", "Paws Vet", "Paws Vet - pets welcome", "Our vets", "5 High Street, Leeds LS1 4AB", "VeterinaryCare"},
		{"sports physio", "Southside Sports Physio", "Sports massage and physio", "Sports injuries", "5 High Street, Leeds LS1 4AB", "MedicalClinic"},
		{"child nursery", "Little Acorns Nursery", "Little Acorns Nursery - book a visit", "Our nursery", "5 High Street, Leeds LS1 4AB", "ChildCare"},
		{"music lessons", "Harmony Music School", "Music lessons for kids", "Music lessons", "5 High Street, Leeds LS1 4AB", "LocalBusiness"},
		{"lawn and garden care", "Green Thumb", "Garden and lawn care", "Garden care", "5 High Street, Leeds LS1 4AB", "LocalBusiness"},
		{"bookkeeper", "Ledger Bookkeeping", "Books kept for small business", "Bookkeeping", "5 High Street, Leeds LS1 4AB", "AccountingService"},
		{"furniture removals", "Smith Furniture Removals", "Furniture removals in Leeds", "Furniture removals", "5 High Street, Leeds LS1 4AB", "MovingCompany"},
		{"trade with no subtype", "Acme", "Acme services", "Acme", "5 High Street, Leeds LS1 4AB", "LocalBusiness"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, shop.LocalStore{}, audittest.Input{Draft: noDraft, PageHTML: tradePage(tc.site, tc.title, tc.h1, tc.address), PageURL: url}, localStoreFor(tc.want, tc.address))
		})
	}
}

// A shop's trade still names its Store subtype, and a page that sells a trade
// no word names is a Store.
func TestLocalStoreShopTrades(t *testing.T) {
	const url = "https://example.test/contact/"
	for _, tc := range []struct {
		name, site, title, h1, want string
		sells                       bool
	}{
		{"book shop", "Harbour Books", "Harbour Books - new and second-hand books", "Visit the bookshop", "BookStore", false},
		{"pet shop", "Paws Pet Supplies", "Paws Pet Supplies - pet food and accessories", "Visit us", "PetStore", false},
		{"sports store", "Peak Sports Store", "Peak Sports Store", "Visit us", "SportingGoodsStore", false},
		{"garden centre", "Fernhill Garden Centre", "Fernhill Garden Centre", "Visit us", "GardenStore", false},
		{"music store", "Riff Music Store", "Riff Music Store", "Visit us", "MusicStore", false},
		{"bare goods in the name", "Spoke and Chain Bicycles", "Visit Spoke and Chain Bicycles", "Visit Spoke and Chain Bicycles", "BikeStore", false},
		{"bare goods, repair service", "Spoke and Chain", "Bike repairs - Spoke and Chain", "Bike repairs", "LocalBusiness", false},
		{"bare goods, lessons", "Volt Computers", "Computer lessons - Volt Computers", "Computer lessons", "LocalBusiness", false},
		{"sells, no trade word", "Acme", "Acme", "Visit us", "Store", true},
		{"does not sell, no trade word", "Acme", "Acme", "Visit us", "LocalBusiness", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := tradePage(tc.site, tc.title, tc.h1, "5 High Street, Leeds LS1 4AB")
			if tc.sells {
				page = audittest.Selling(page)
			}
			audittest.Expect(t, shop.LocalStore{}, audittest.Input{Draft: noDraft, PageHTML: page, PageURL: url}, localStoreFor(tc.want, "5 High Street, Leeds LS1 4AB"))
		})
	}
}

// The business-type check names the same service types.
func TestBusinessTypeServiceTrade(t *testing.T) {
	page := tradePage("Moorehead Dentistry", "Dental Office Fees - Moorehead Dentistry - 3 Locations in Ohio", "Office Fees", "10427 Kenwood Road, Cincinnati, OH 45242")
	audittest.Expect(t, shop.BusinessType{}, audittest.Input{Draft: business, PageHTML: page, PageURL: "https://example.test/fees/"}, []audit.Issue{
		{Path: "$", Severity: audit.Warning, Message: `the page shows the street address "10427 Kenwood Road, Cincinnati, OH 45242": type this Organization Dentist`},
	})
}
