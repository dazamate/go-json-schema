package schema

import "testing"

func TestIdentityURLHelpers(t *testing.T) {
	cases := map[URL]URL{
		Wikipedia("Search engine optimization"):         "https://en.wikipedia.org/wiki/Search_engine_optimization",
		WikipediaLang("de", "Suchmaschinenoptimierung"): "https://de.wikipedia.org/wiki/Suchmaschinenoptimierung",
		Wikidata("Q180711"):                             "https://www.wikidata.org/wiki/Q180711",
		DBpedia("Fountain Hills, Arizona"):              "http://dbpedia.org/resource/Fountain_Hills,_Arizona",
		ProductOntology("Web design"):                   "http://productontology.org/id/Web_design",
		KGSearch("internet marketing", "/g/122hcnps"):   "https://www.google.com/search?q=internet+marketing&kponly&kgmid=/g/122hcnps",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func TestGoogleMachineIDIdentifier(t *testing.T) {
	got := mustMarshal(t, GoogleMachineID("/m/0f1w9"))
	want := `{"@type":"PropertyValue","propertyID":"GoogleMachineID","value":"kg:/m/0f1w9"}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestEntityThingCarriesEverySource(t *testing.T) {
	e := Entity{
		Name:      "seo",
		Wikipedia: "Search_engine_optimization",
		Wikidata:  "Q180711",
		KGMID:     "/m/019qb_",
		DBpedia:   "Search_engine_optimization",
		SameAs:    []string{"https://example.com/seo"},
	}
	got := mustMarshal(t, e.Thing())
	want := `{"@type":"Thing","identifier":{"@type":"PropertyValue","propertyID":"GoogleMachineID","value":"kg:/m/019qb_"},"name":"seo","sameAs":["https://en.wikipedia.org/wiki/Search_engine_optimization","https://www.wikidata.org/wiki/Q180711","https://www.google.com/search?q=seo&kponly&kgmid=/m/019qb_","http://dbpedia.org/resource/Search_engine_optimization","https://example.com/seo"]}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestEntityTypeAndID(t *testing.T) {
	e := Entity{Type: "State", ID: string(Wikipedia("Arizona")), Name: "Arizona"}
	got := mustMarshal(t, e.Thing())
	want := `{"@type":"State","@id":"https://en.wikipedia.org/wiki/Arizona","name":"Arizona"}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestEntityPerSourceSplitsOneThingPerURL(t *testing.T) {
	e := Entity{Name: "Fountain Hills", Wikipedia: "Fountain_Hills,_Arizona", KGMID: "/m/01z50zj"}
	got := mustMarshal(t, e.PerSource())
	want := `[{"@type":"Thing","name":"Fountain Hills","sameAs":"https://en.wikipedia.org/wiki/Fountain_Hills,_Arizona"},{"@type":"Thing","name":"Fountain Hills","sameAs":"https://www.google.com/search?q=Fountain+Hills&kponly&kgmid=/m/01z50zj"}]`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestEntitiesBuildsValues(t *testing.T) {
	vs := Entities(Entity{Name: "a"}, Entity{Name: "b"})
	if len(vs) != 2 {
		t.Fatalf("len = %d", len(vs))
	}
}
