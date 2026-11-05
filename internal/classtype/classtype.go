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
// those, not the general class. The venue, business and non-US place classes
// (hotel, museum, airport, local government area of Victoria, county of Ohio,
// province of Canada and the like) were found by counting the P31 classes of
// items in AU, CA, GB, IE, IN, NZ and US and checked on 2026-11-06.
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
	{"School", false, []string{
		"Q3914",   // school
		"Q9842",   // primary school
		"Q159334", // secondary school
	}},
	{"Hospital", false, []string{
		"Q16917",    // hospital
		"Q7257872",  // public hospital
		"Q4284971",  // private hospital
		"Q61451856", // rural hospital
		"Q2387974",  // military hospital
		"Q7048982",  // non-profit hospital
	}},
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
	{"Hotel", false, []string{
		"Q27686",    // hotel
		"Q617839",   // apartment hotel
		"Q64995589", // country house hotel
		"Q259618",   // boutique hotel
		"Q16659419", // railway hotel
	}},
	{"Hostel", false, []string{
		"Q654772", // hostel
	}},
	{"Motel", false, []string{
		"Q216212", // motel
	}},
	{"LodgingBusiness", false, []string{
		"Q2460422",  // guest house
		"Q56439882", // lodge
	}},
	{"Restaurant", false, []string{
		"Q11707",   // restaurant
		"Q1226318", // diner
		"Q3109696", // steakhouse
	}},
	{"FastFoodRestaurant", false, []string{
		"Q1751429", // fast food restaurant
	}},
	{"IceCreamShop", false, []string{
		"Q1311064", // ice cream parlor
	}},
	{"Museum", false, []string{
		"Q33506",     // museum
		"Q207694",    // art museum
		"Q2087181",   // historic house museum
		"Q16735822",  // history museum
		"Q115154402", // independent museum
		"Q115154345", // local authority museum
	}},
	{"Airport", false, []string{
		"Q1248784", // airport
		"Q644371",  // international airport
		"Q837800",  // domestic airport
		"Q2138424", // regional airport
	}},
	{"TrainStation", false, []string{
		"Q55488",   // railway station
		"Q7886778", // union station
		"Q55485",   // dead-end railway station
	}},
	{"SubwayStation", false, []string{
		"Q928830", // metro station
		"Q55491",  // underground railway station
	}},
	{"ShoppingCenter", false, []string{
		"Q11315",    // shopping center
		"Q31374404", // shopping mall
		"Q7624088",  // strip mall
		"Q39658032", // outdoor shopping centre
		"Q54927709", // outlet mall
		"Q97495053", // shopping arcade
		"Q1824486",  // lifestyle center
	}},
	{"Park", false, []string{
		"Q22698",   // park
		"Q22746",   // urban park
		"Q46169",   // national park
		"Q6063204", // regional park
		"Q1761072", // state park
		"Q350723",  // country park
	}},
	{"Zoo", false, []string{
		"Q43501",   // zoo
		"Q642682",  // safari park
		"Q1711697", // wildlife park
	}},
	{"Aquarium", false, []string{
		"Q2281788", // public aquarium
	}},
	{"StadiumOrArena", false, []string{
		"Q483110",   // stadium
		"Q595452",   // baseball venue
		"Q1154710",  // association football venue
		"Q45290083", // rugby league venue
		"Q15303456", // rugby union venue
		"Q37985280", // American football venue
	}},
	{"Library", false, []string{
		"Q7075",     // library
		"Q28564",    // public library
		"Q11396180", // library branch
		"Q1438040",  // research library
		"Q385994",   // special library
		"Q1076099",  // school library
	}},
	{"Beach", false, []string{
		"Q40080",    // beach
		"Q18751767", // sand beach
		"Q11959973", // bathing beach
		"Q847935",   // nude beach
		"Q7900097",  // urban beach
		"Q35800652", // state beach
	}},
	{"Church", false, []string{
		"Q16970",    // church building
		"Q108325",   // chapel
		"Q56242063", // Protestant church building
		"Q317557",   // parish church
		"Q2977",     // cathedral
	}},
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
		"Q3336843",  // constituent country of the United Kingdom
	}},
	{"State", false, []string{
		"Q107390",   // federated state
		"Q5852411",  // state of Australia
		"Q35657",    // U.S. state
		"Q34876",    // province
		"Q10864048", // first-level administrative division
		"Q15149663", // state of Mexico
		"Q14192199", // territory of Australia
		"Q14192234", // mainland territory of Australia
		"Q11828004", // province of Canada
		"Q9357527",  // territory of Canada
		"Q12443800", // state of India
		"Q467745",   // union territory of India
	}},
	{"City", false, []string{
		"Q515",       // city
		"Q1637706",   // million city
		"Q200250",    // metropolis
		"Q1549591",   // big city
		"Q5119",      // capital city
		"Q3957",      // town
		"Q7930989",   // city or town
		"Q1093829",   // city in the United States
		"Q51929311",  // largest city
		"Q22865",     // independent city of Germany
		"Q21010817",  // city of Pennsylvania
		"Q13218357",  // city of California
		"Q63440326",  // city of Oregon
		"Q15221304",  // first-class city
		"Q15221310",  // second-class city
		"Q15221370",  // third-class city
		"Q15221373",  // fourth-class city
		"Q1266818",   // independent city in the United States
		"Q604886",    // charter city of the United States
		"Q138164278", // city in Arizona
		"Q2974552",   // city in New Jersey
		"Q15063611",  // city in the state of New York
		"Q110071194", // city of Illinois
		"Q139903024", // city of Michigan
		"Q110292529", // city of Minnesota
		"Q132680629", // city of North Carolina
		"Q134844639", // city of North Dakota
		"Q108569710", // city of Ohio
		"Q138010797", // city of Texas
		"Q111874502", // city of West Virginia
		"Q15221256",  // city of Wisconsin
		"Q15127012",  // town in the United States
		"Q2154459",   // New England town
		"Q138164972", // town in Arizona
		"Q135396264", // town in California
		"Q135396321", // town in Indiana
		"Q139961091", // town of Maryland
		"Q7829892",   // town of New Jersey
		"Q106071004", // town of New York
		"Q110256876", // town of Pennsylvania
		"Q15411644",  // town of Virginia
		"Q132172647", // town of West Virginia
		"Q140044940", // statutory town of Colorado
		"Q55430416",  // town in Alberta
		"Q52132873",  // town in New Brunswick
		"Q6643756",   // town in Saskatchewan
		"Q131905118", // town in Manitoba
		"Q55440238",  // city in Alberta
		"Q60458065",  // city in British Columbia
		"Q27676416",  // city or town of Quebec
		"Q21507383",  // provincial or territorial capital city in Canada
		"Q11271835",  // state capital
		"Q141763519", // city of the United Kingdom
		"Q1867820",   // city status in the United Kingdom
		"Q100503226", // town of the United Kingdom
		"Q18511725",  // market town
		"Q1357964",   // county town
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
		"Q1426035",   // local government area of Victoria
		"Q55558200",  // local government area of New South Wales
		"Q55593624",  // local government area of Queensland
		"Q55558027",  // local government area of South Australia
		"Q55687066",  // local government area of Tasmania
		"Q55557858",  // local government area of Western Australia
		"Q55671590",  // local government area of the Northern Territory
		"Q26907711",  // district of the Australian Capital Territory
		"Q216712",    // region of New Zealand
		"Q2881272",   // district of New Zealand
		"Q15063160",  // administrative territorial entity of New Zealand
		"Q179872",    // county of Ireland
		"Q202156",    // province of Ireland
		"Q85637361",  // local government county in Ireland
		"Q1187580",   // non-metropolitan district
		"Q180673",    // ceremonial county of England
		"Q769603",    // non-metropolitan county
		"Q21272231",  // county council area
		"Q2560047",   // administrative county
		"Q15979307",  // principal area
		"Q1006876",   // borough in the United Kingdom
		"Q2249932",   // district of Wales
		"Q211690",    // London borough
		"Q21503295",  // district with city status
		"Q48091",     // region of England
		"Q1136601",   // unitary authority area
		"Q15060255",  // council area
		"Q1115575",   // civil parish
		"Q1149652",   // district of India
		"Q63076981",  // district of Kerala
		"Q2775236",   // community development block in India
		"Q18918041",  // municipality of West Bengal
		"Q2991484",   // county of Canada
		"Q18810091",  // census division of Canada
		"Q15284",     // municipality
		"Q27676428",  // municipality
		"Q3327874",   // rural municipality of Canada
		"Q55774719",  // township municipality in Ontario
		"Q1388464",   // regional municipality in Canada
		"Q408804",    // borough of New York City
		"Q13410400",  // county of Alabama
		"Q13217186",  // county of Arizona
		"Q11774062",  // county of Arkansas
		"Q13410403",  // county of Colorado
		"Q13410411",  // county of Delaware
		"Q13410428",  // county of Georgia
		"Q13410433",  // county of Idaho
		"Q13410438",  // county of Indiana
		"Q11774149",  // county of Iowa
		"Q13410444",  // county of Kansas
		"Q13410447",  // county of Kentucky
		"Q13410454",  // county of Maine
		"Q13410464",  // county of Maryland
		"Q13410485",  // county of Massachusetts
		"Q13410496",  // county of Michigan
		"Q12262532",  // county of Minnesota
		"Q13410508",  // county of Mississippi
		"Q13410520",  // county of Missouri
		"Q13414358",  // county of Montana
		"Q13414361",  // county of Nebraska
		"Q13414753",  // county of New Hampshire
		"Q13414754",  // county of New Jersey
		"Q13414755",  // county of New Mexico
		"Q13414757",  // county of New York
		"Q13414758",  // county of North Carolina
		"Q12178655",  // county of North Dakota
		"Q13414759",  // county of Ohio
		"Q13414760",  // county of Oklahoma
		"Q13414761",  // county of Oregon
		"Q13414763",  // county of Pennsylvania
		"Q13414764",  // county of Rhode Island
		"Q13414765",  // county of South Carolina
		"Q12037308",  // county of South Dakota
		"Q13415366",  // county of Tennessee
		"Q11774097",  // county of Texas
		"Q13415365",  // county of Utah
		"Q13415367",  // county of Vermont
		"Q13415368",  // county of Virginia
		"Q13415369",  // county of Washington
		"Q13415370",  // county of West Virginia
		"Q12178928",  // county of Wisconsin
		"Q13415371",  // county of Wyoming
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
		"Q188509",   // suburb
		"Q3257686",  // locality
		"Q45242174", // gazetted locality of Victoria
		"Q7632428",  // suburb of Canberra
		"Q23010647", // suburb of Perth
		"Q674950",   // residential area
		"Q738570",   // central business district
		"Q702492",   // urban area
		"Q55237813", // village of New York
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
