package schema

import (
	"encoding/json"
	"reflect"
	"testing"
)

// modelPagePairs is every (type, property) pair used in the JSON-LD of the
// two model pages for over-the-top schema: digitaleer.com/fountain-hills-seo
// and salterrasite.com (fetched 2026-10-23).
var modelPagePairs = [][2]string{
	{"Answer", "text"},
	{"BreadcrumbList", "itemListElement"},
	{"BreadcrumbList", "name"},
	{"City", "addressRegion"},
	{"City", "geo"},
	{"City", "name"},
	{"ContactPoint", "areaServed"},
	{"ContactPoint", "availableLanguage"},
	{"ContactPoint", "contactType"},
	{"ContactPoint", "email"},
	{"ContactPoint", "telephone"},
	{"Country", "description"},
	{"Country", "identifier"},
	{"Country", "name"},
	{"Country", "sameAs"},
	{"Country", "url"},
	{"FAQPage", "mainEntity"},
	{"GeoCoordinates", "description"},
	{"GeoCoordinates", "latitude"},
	{"GeoCoordinates", "longitude"},
	{"GeoCoordinates", "name"},
	{"GeoCoordinates", "postalCode"},
	{"ImageObject", "author"},
	{"ImageObject", "contentLocation"},
	{"ImageObject", "creator"},
	{"ImageObject", "description"},
	{"ImageObject", "height"},
	{"ImageObject", "name"},
	{"ImageObject", "url"},
	{"ImageObject", "width"},
	{"ListItem", "item"},
	{"ListItem", "position"},
	{"Offer", "itemOffered"},
	{"OfferCatalog", "additionalType"},
	{"OfferCatalog", "itemListElement"},
	{"OfferCatalog", "name"},
	{"OfferCatalog", "url"},
	{"Organization", "address"},
	{"Organization", "alternateName"},
	{"Organization", "areaServed"},
	{"Organization", "contactPoint"},
	{"Organization", "description"},
	{"Organization", "founder"},
	{"Organization", "foundingDate"},
	{"Organization", "identifier"},
	{"Organization", "knowsAbout"},
	{"Organization", "logo"},
	{"Organization", "name"},
	{"Organization", "sameAs"},
	{"Organization", "url"},
	{"Person", "additionalName"},
	{"Person", "additionalType"},
	{"Person", "address"},
	{"Person", "alternateName"},
	{"Person", "contactPoint"},
	{"Person", "description"},
	{"Person", "disambiguatingDescription"},
	{"Person", "email"},
	{"Person", "familyName"},
	{"Person", "gender"},
	{"Person", "givenName"},
	{"Person", "image"},
	{"Person", "jobTitle"},
	{"Person", "knowsAbout"},
	{"Person", "knowsLanguage"},
	{"Person", "memberOf"},
	{"Person", "name"},
	{"Person", "potentialAction"},
	{"Person", "sameAs"},
	{"Person", "telephone"},
	{"Person", "url"},
	{"Person", "workLocation"},
	{"Person", "worksFor"},
	{"Place", "address"},
	{"Place", "name"},
	{"Place", "url"},
	{"PostalAddress", "addressCountry"},
	{"PostalAddress", "addressLocality"},
	{"PostalAddress", "addressRegion"},
	{"PostalAddress", "postalCode"},
	{"PostalAddress", "streetAddress"},
	{"ProfessionalService", "additionalProperty"},
	{"ProfessionalService", "additionalType"},
	{"ProfessionalService", "address"},
	{"ProfessionalService", "alternateName"},
	{"ProfessionalService", "areaServed"},
	{"ProfessionalService", "award"},
	{"ProfessionalService", "contactPoint"},
	{"ProfessionalService", "currenciesAccepted"},
	{"ProfessionalService", "description"},
	{"ProfessionalService", "disambiguatingDescription"},
	{"ProfessionalService", "email"},
	{"ProfessionalService", "founder"},
	{"ProfessionalService", "foundingDate"},
	{"ProfessionalService", "geo"},
	{"ProfessionalService", "hasMap"},
	{"ProfessionalService", "hasOfferCatalog"},
	{"ProfessionalService", "image"},
	{"ProfessionalService", "knowsAbout"},
	{"ProfessionalService", "logo"},
	{"ProfessionalService", "mainEntityOfPage"},
	{"ProfessionalService", "name"},
	{"ProfessionalService", "openingHours"},
	{"ProfessionalService", "paymentAccepted"},
	{"ProfessionalService", "priceRange"},
	{"ProfessionalService", "provider"},
	{"ProfessionalService", "sameAs"},
	{"ProfessionalService", "telephone"},
	{"ProfessionalService", "url"},
	{"PropertyValue", "propertyID"},
	{"PropertyValue", "value"},
	{"Question", "acceptedAnswer"},
	{"Question", "name"},
	{"Rating", "bestRating"},
	{"Rating", "ratingValue"},
	{"SearchAction", "query-input"},
	{"SearchAction", "target"},
	{"Service", "description"},
	{"Service", "name"},
	{"Service", "sameAs"},
	{"Service", "url"},
	{"State", "description"},
	{"State", "identifier"},
	{"State", "name"},
	{"State", "sameAs"},
	{"State", "url"},
	{"Thing", "description"},
	{"Thing", "identifier"},
	{"Thing", "name"},
	{"Thing", "sameAs"},
	{"Thing", "url"},
	{"WebPage", "about"},
	{"WebPage", "hasPart"},
	{"WebPage", "isPartOf"},
	{"WebPage", "mainEntity"},
	{"WebPage", "mentions"},
	{"WebPage", "name"},
	{"WebPage", "publisher"},
	{"WebPage", "url"},
	{"WebSite", "about"},
	{"WebSite", "alternateName"},
	{"WebSite", "description"},
	{"WebSite", "disambiguatingDescription"},
	{"WebSite", "hasPart"},
	{"WebSite", "identifier"},
	{"WebSite", "keywords"},
	{"WebSite", "mentions"},
	{"WebSite", "name"},
	{"WebSite", "potentialAction"},
	{"WebSite", "publisher"},
	{"WebSite", "url"},
}

// extraOnly are the pairs that schema.org does not list, so they go through
// Node.Extra instead of a typed field.
var extraOnly = map[[2]string]bool{
	{"City", "addressRegion"}:           true,
	{"ProfessionalService", "provider"}: true,
	{"SearchAction", "query-input"}:     true,
}

func encodedKeys(t *testing.T, typeName string) map[string]any {
	t.Helper()
	node, ok := New(typeName)
	if !ok {
		t.Fatalf("no generated type %s", typeName)
	}
	fillAll(reflect.ValueOf(node).Elem())
	b, err := Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}

func TestModelPagePropertiesHaveTypedFields(t *testing.T) {
	for _, pair := range modelPagePairs {
		if extraOnly[pair] {
			continue
		}
		if _, ok := encodedKeys(t, pair[0])[pair[1]]; !ok {
			t.Errorf("%s has no typed field for %s", pair[0], pair[1])
		}
	}
}

func TestModelPageOffDomainPropertiesWorkThroughExtra(t *testing.T) {
	for pair := range extraOnly {
		if _, ok := encodedKeys(t, pair[0])[pair[1]]; ok {
			t.Errorf("%s.%s now has a typed field; drop it from extraOnly", pair[0], pair[1])
		}
		node, _ := New(pair[0])
		reflect.ValueOf(node).Elem().FieldByName("Extra").Set(reflect.ValueOf(map[string]Values{pair[1]: Texts("x")}))
		b, err := Marshal(node)
		if err != nil {
			t.Fatal(err)
		}
		var obj map[string]any
		if err := json.Unmarshal(b, &obj); err != nil {
			t.Fatal(err)
		}
		if obj[pair[1]] != "x" {
			t.Errorf("%s.%s via Extra: got %s", pair[0], pair[1], b)
		}
	}
}
