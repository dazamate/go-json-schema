// Package classtype maps Wikidata classes to schema.org types. The research
// tools type entity nodes with it, and build_jsonld checks Thing nodes against
// it, so the two never disagree.
package classtype

import "slices"

// table maps Wikidata classes to the schema.org type of their instances,
// most specific type first: an item takes the first type that one of its own
// P31 (instance of) classes is listed under. Only P31 counts. Walking P279
// (subclass of) upward reaches classes like abstract entity or class, whose
// schema.org equivalents (Intangible, Class) are wrong for an entity, and an
// item whose P31 is a class of classes (a wedding is a "key event", a bride
// a "term for a woman") is a general idea, which stays a Thing. So the table
// never maps to Event, Product, CreativeWork, DefinedTerm or Intangible.
//
// An item with a P279 claim is itself a class: a kind of thing, not one
// (moving company is an "instance of company" but a subclass of business),
// so it stays a Thing too. Kinds marks the types whose Wikidata items are
// always classes, languages and occupations, which take their type anyway.
// Every ID was checked against its Wikidata label on 2026-10-30, and the
// state-level US classes (city of Pennsylvania, county of California, United
// States federal agency) on 2026-10-31: Wikidata gives US places and agencies
// those, not the general class.
var table = []struct {
	Type    string
	Kinds   bool
	Classes []string
}{
	{"Person", false, []string{"Q5"}}, // human
	{"Occupation", true, []string{
		"Q28640",    // profession
		"Q12737077", // occupation
		"Q66715801", // musical profession
	}},
	{"Language", true, []string{
		"Q34770",    // language
		"Q33742",    // natural language
		"Q1288568",  // modern language
		"Q20162172", // human language
		"Q152559",   // macrolanguage
		"Q941501",   // language group
		"Q399495",   // standard language
		"Q33384",    // dialect
		"Q25295",    // language family
		"Q34228",    // sign language
		"Q45762",    // dead language
		"Q436240",   // ancient language
		"Q1149626",  // written language
	}},
	{"Brand", false, []string{"Q431289"}}, // brand
	{"CollegeOrUniversity", false, []string{
		"Q3918",     // university
		"Q875538",   // public university
		"Q62078547", // public research university
	}},
	{"School", false, []string{"Q3914"}},    // school
	{"Hospital", false, []string{"Q16917"}}, // hospital
	{"GovernmentOrganization", false, []string{
		"Q327333",   // government agency
		"Q20857065", // United States federal agency
		"Q1752939",  // independent agency of the United States government
		"Q74330265", // state department of motor vehicles
		"Q97010948", // state department of transportation of the United States
		"Q732717",   // law enforcement agency
		"Q2587942",  // housing ministry
		"Q1364302",  // ministry of agriculture
		"Q1898583",  // marine force
		"Q189445",   // bicameral legislature
	}},
	{"DefenceEstablishment", false, []string{
		"Q245016", // military base
		"Q695850", // airbase
	}},
	{"MovingCompany", false, []string{"Q1757263"}},         // moving company
	{"NewsMediaOrganization", false, []string{"Q1153191"}}, // online newspaper
	{"NGO", false, []string{"Q163740"}},                    // nonprofit organization
	{"MusicGroup", false, []string{"Q215380"}},             // musical group
	{"SportsTeam", false, []string{"Q12973014"}},           // sports team
	{"Organization", false, []string{
		"Q7835189", // transit district
		"Q43229",   // organization
		"Q4830453", // business
		"Q783794",  // company
		"Q891723",  // public company
		"Q2085381", // publishing house
		"Q6881511", // enterprise
	}},
	{"Country", false, []string{
		"Q6256",     // country
		"Q3624078",  // sovereign state
		"Q112099",   // island country
		"Q179164",   // unitary state
		"Q15634554", // state with limited recognition
		"Q3024240",  // historical country
	}},
	{"State", false, []string{
		"Q107390",   // federated state
		"Q5852411",  // state of Australia
		"Q35657",    // U.S. state
		"Q34876",    // province
		"Q10864048", // first-level administrative division
		"Q15149663", // state of Mexico
	}},
	{"City", false, []string{
		"Q515",      // city
		"Q1637706",  // million city
		"Q200250",   // metropolis
		"Q1549591",  // big city
		"Q5119",     // capital city
		"Q3957",     // town
		"Q7930989",  // city or town
		"Q1093829",  // city in the United States
		"Q51929311", // largest city
		"Q22865",    // independent city of Germany
		"Q21010817", // city of Pennsylvania
		"Q13218357", // city of California
		"Q63440326", // city of Oregon
	}},
	{"SchoolDistrict", false, []string{"Q15726209"}}, // school district in the United States
	{"AdministrativeArea", false, []string{
		"Q56061",     // administrative territorial entity
		"Q1867183",   // local government area of Australia
		"Q28575",     // county
		"Q13220204",  // second-level administrative division
		"Q13221722",  // third-level administrative division
		"Q13212489",  // county of California
		"Q131427665", // charter county of California
		"Q13414369",  // county of Nevada
		"Q11812346",  // county of Illinois
		"Q13410422",  // county of Florida
		"Q13414354",  // county of Connecticut
		"Q13410431",  // county of Hawaii
	}},
	{"Bridge", false, []string{
		"Q7814330",  // toll bridge
		"Q19941854", // bridge group
	}},
	{"LandmarksOrHistoricalBuildings", false, []string{
		"Q1564373", // mission station
		"Q57821",   // fortification
	}},
	{"Continent", false, []string{"Q5107"}},        // continent
	{"Mountain", false, []string{"Q8502"}},         // mountain
	{"LakeBodyOfWater", false, []string{"Q23397"}}, // lake
	{"RiverBodyOfWater", false, []string{"Q4022"}}, // river
	{"SeaBodyOfWater", false, []string{"Q165"}},    // sea
	{"OceanBodyOfWater", false, []string{"Q9430"}}, // ocean
	{"BodyOfWater", false, []string{"Q37901"}},     // strait
	{"Landform", false, []string{
		"Q46831",  // mountain range
		"Q23442",  // island
		"Q271669", // landform
		"Q39816",  // valley
		"Q133056", // mountain pass
		"Q34763",  // peninsula
	}},
	{"Place", false, []string{
		"Q82794",    // region
		"Q2221906",  // geographic location
		"Q17334923", // physical location
		"Q1620908",  // historical region
		"Q123705",   // neighborhood
		"Q486972",   // human settlement
		"Q532",      // village
		"Q1768043",  // metropolitan statistical area
		"Q34442",    // road (schema.org has no Road)
		"Q46622",    // controlled-access highway
		"Q498162",   // census-designated place in the United States
		"Q52154375", // district or neighborhood of Los Angeles
	}},
}

// Of is the schema.org type for an item with these P31 classes, or "" (a
// Thing) when none is in the table. isClass reports that the item has a P279
// (subclass of) claim.
func Of(p31 []string, isClass bool) string {
	for _, t := range table {
		if isClass && !t.Kinds {
			continue
		}
		if slices.ContainsFunc(p31, func(c string) bool { return slices.Contains(t.Classes, c) }) {
			return t.Type
		}
	}
	return ""
}
