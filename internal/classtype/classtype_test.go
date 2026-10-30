package classtype

import (
	"slices"
	"testing"

	"github.com/dazamate/go-json-schema/schema"
)

// TestOf types items by their P31 classes as Wikidata gave them on
// 2026-10-30, preferred rank first: instances get their schema.org type, and
// general ideas, whose P31 is a class of classes or absent, stay a Thing.
func TestOf(t *testing.T) {
	for _, tc := range []struct {
		name    string
		p31     []string
		isClass bool // has P279
		want    string
	}{
		{"Cantonese", []string{"Q33742", "Q1288568", "Q2748296"}, true, "Language"},
		{"English", []string{"Q33742", "Q1288568", "Q34770", "Q152559", "Q399495", "Q1097949", "Q20162172"}, true, "Language"},
		{"Mandarin", []string{"Q34770"}, true, "Language"},
		{"Chinese", []string{"Q152559", "Q1288568", "Q941501"}, true, "Language"},
		{"Standard Chinese", []string{"Q399495", "Q317623", "Q34770", "Q1288568", "Q250858"}, true, "Language"},
		{"Yue Chinese", []string{"Q34770", "Q1288568", "Q2748296"}, true, "Language"},
		{"Sydney", []string{"Q515", "Q200250", "Q1549591", "Q51929311", "Q11271835", "Q208511", "Q1066984"}, false, "City"},
		{"People's Republic of China", []string{"Q3624078", "Q6256", "Q842112", "Q859563", "Q1520223", "Q465613", "Q118365", "Q849866", "Q185145", "Q10711424"}, false, "Country"},
		{"East Asia", []string{"Q82794"}, false, "Place"},
		{"Central Coast", []string{"Q82794"}, false, "Place"},
		{"Blue Mountains", []string{"Q46831", "Q82794", "Q1640361"}, false, "Landform"},
		{"master of ceremonies", []string{"Q28640", "Q12737077", "Q214339"}, true, "Occupation"},
		{"photographer", []string{"Q12737077", "Q108300140"}, true, "Occupation"},
		{"disc jockey", []string{"Q66715801", "Q12737077"}, true, "Occupation"},
		// general ideas, held on P31 alone (isClass false)
		{"wedding", []string{"Q2245405", "Q82821"}, false, ""},                              // key event, tradition
		{"bride", []string{"Q954007", "Q135929693"}, false, ""},                             // affinity, term for a woman
		{"drink", []string{"Q19861951"}, false, ""},                                         // type of food or dish
		{"dance", []string{"Q45971958", "Q47728"}, false, ""},                               // performing arts genre, hobby
		{"toast", []string{"Q187931"}, false, ""},                                           // phrase
		{"Chinese Australians", []string{"Q108788952"}, false, ""},                          // ethnic group by residency
		{"marriage", []string{"Q1938208", "Q11666901", "Q108586636", "Q178706"}, false, ""}, // legal institution, nominal kinship, form of event, institution
		{"ceremony", []string{"Q82821"}, false, ""},                                         // tradition
		{"Chinese marriage", []string{"Q74817647"}, false, ""},                              // aspect in a geographic region
		{"red envelope", nil, false, ""},
		{"tea ceremony", nil, false, ""},
		// a kind of company: instance of company and service provider, subclass of business
		{"moving company", []string{"Q783794", "Q2169973"}, true, ""},
		{"a business", []string{"Q4830453"}, false, "Organization"},
		// US places, agencies and landmarks, by their first P31 class as
		// Wikidata gave it on 2026-10-31
		{"Philadelphia", []string{"Q21010817"}, false, "City"},
		{"Santa Monica", []string{"Q13218357"}, false, "City"},
		{"Medford, Oregon", []string{"Q63440326"}, false, "City"},
		{"Federal Motor Carrier Safety Administration", []string{"Q20857065"}, false, "GovernmentOrganization"},
		{"Surface Transportation Board", []string{"Q1752939"}, false, "GovernmentOrganization"},
		{"Texas Department of Motor Vehicles", []string{"Q74330265"}, false, "GovernmentOrganization"},
		{"District Department of Transportation", []string{"Q97010948"}, false, "GovernmentOrganization"},
		{"Georgia Department of Public Safety", []string{"Q732717"}, false, "GovernmentOrganization"},
		{"HUD", []string{"Q2587942"}, false, "GovernmentOrganization"},
		{"California Department of Food and Agriculture", []string{"Q1364302"}, false, "GovernmentOrganization"},
		{"United States Marine Corps", []string{"Q1898583"}, false, "GovernmentOrganization"},
		{"United States Congress", []string{"Q189445"}, false, "GovernmentOrganization"},
		{"San Diego County", []string{"Q131427665", "Q13212489"}, false, "AdministrativeArea"},
		{"Marin County", []string{"Q13212489"}, false, "AdministrativeArea"},
		{"Clark County, Nevada", []string{"Q13414369"}, false, "AdministrativeArea"},
		{"Cook County", []string{"Q11812346"}, false, "AdministrativeArea"},
		{"Miami-Dade County", []string{"Q13410422"}, false, "AdministrativeArea"},
		{"Fairfield County", []string{"Q13414354"}, false, "AdministrativeArea"},
		{"Honolulu County", []string{"Q13410431"}, false, "AdministrativeArea"},
		{"Los Angeles Unified School District", []string{"Q15726209"}, false, "SchoolDistrict"},
		{"Baja California", []string{"Q15149663"}, false, "State"},
		{"Greater Houston", []string{"Q1768043"}, false, "Place"},
		{"Interstate 5", []string{"Q34442"}, false, "Place"},
		{"Merritt Parkway", []string{"Q46622"}, false, "Place"},
		{"The Woodlands", []string{"Q498162"}, false, "Place"},
		{"Koreatown", []string{"Q52154375"}, false, "Place"},
		{"San Fernando Valley", []string{"Q39816"}, false, "Landform"},
		{"Bozeman Pass", []string{"Q133056"}, false, "Landform"},
		{"San Francisco Peninsula", []string{"Q34763"}, false, "Landform"},
		{"Golden Gate", []string{"Q37901"}, false, "BodyOfWater"},
		{"Alamo Mission", []string{"Q1564373", "Q57821"}, false, "LandmarksOrHistoricalBuildings"},
		{"Naval Base San Diego", []string{"Q245016"}, false, "DefenceEstablishment"},
		{"MCAS Miramar", []string{"Q695850"}, false, "DefenceEstablishment"},
		{"San Diego State University", []string{"Q875538"}, false, "CollegeOrUniversity"},
		{"UC San Diego", []string{"Q62078547"}, false, "CollegeOrUniversity"},
		{"College Hunks Hauling Junk", []string{"Q1757263"}, false, "MovingCompany"},
		{"Dumbarton Bridge", []string{"Q7814330"}, false, "Bridge"},
		{"San Francisco–Oakland Bay Bridge", []string{"Q19941854"}, false, "Bridge"},
		{"San Francisco Municipal Railway", []string{"Q7835189"}, false, "Organization"},
		{"CNET", []string{"Q1153191"}, false, "NewsMediaOrganization"},
		// a dated happening stays a Thing: the table never maps to Event
		{"Fiesta San Antonio", []string{"Q132241"}, false, ""},
		{"Hurricane Helene", []string{"Q34439356"}, false, ""},
		// the kind itself, a subclass, stays a Thing
		{"moving company (the kind)", []string{"Q1757263"}, true, ""},
	} {
		if got := Of(tc.p31, tc.isClass); got != tc.want {
			t.Errorf("%s: Of(%v, %v) = %q, want %q", tc.name, tc.p31, tc.isClass, got, tc.want)
		}
	}
}

// TestTable checks the table: every type is a schema.org type, none is
// one an entity found in text should never take, and no class is listed twice.
func TestTable(t *testing.T) {
	seen := map[string]string{}
	for _, row := range table {
		if _, ok := schema.New(row.Type); !ok {
			t.Errorf("%s is not a schema.org type", row.Type)
		}
		if slices.Contains([]string{"Thing", "Event", "Product", "CreativeWork", "DefinedTerm", "Intangible", "Class"}, row.Type) {
			t.Errorf("%s must not be in the table: a general idea stays a Thing", row.Type)
		}
		for _, c := range row.Classes {
			if prev, dup := seen[c]; dup {
				t.Errorf("%s is listed under %s and %s", c, prev, row.Type)
			}
			seen[c] = row.Type
		}
	}
}
