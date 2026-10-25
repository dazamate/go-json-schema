package schema

import "strings"

// Wikipedia returns the English Wikipedia URL for an article title.
func Wikipedia(title string) URL { return WikipediaLang("en", title) }

// WikipediaLang returns the Wikipedia URL for an article title in the given
// language edition.
func WikipediaLang(lang, title string) URL {
	return URL("https://" + lang + ".wikipedia.org/wiki/" + underscore(title))
}

// Wikidata returns the URL of a Wikidata item, e.g. Q180711.
func Wikidata(qid string) URL { return URL("https://www.wikidata.org/wiki/" + qid) }

// DBpedia returns the DBpedia resource URL for an article title.
func DBpedia(title string) URL { return URL("http://dbpedia.org/resource/" + underscore(title)) }

// ProductOntology returns the productontology.org class URL for a term,
// for use in additionalType.
func ProductOntology(term string) URL {
	return URL("http://productontology.org/id/" + underscore(term))
}

// KGSearch returns a Google search URL pinned to a Knowledge Graph entity by
// its machine ID, e.g. /m/019qb_ or /g/122hcnps.
func KGSearch(mid string) URL {
	return URL("https://www.google.com/search?kgmid=" + mid)
}

// GoogleMachineID returns an identifier naming a Google Knowledge Graph
// machine ID, encoded as {"propertyID":"GoogleMachineID","value":"kg:<mid>"}.
func GoogleMachineID(mid string) *PropertyValue {
	pv := &PropertyValue{}
	pv.PropertyID = Texts("GoogleMachineID")
	pv.Value = Texts("kg:" + mid)
	return pv
}

// WikidataID returns an identifier naming a Wikidata item, encoded as
// {"propertyID":"Wikidata","value":"<qid>"}.
func WikidataID(qid string) *PropertyValue {
	pv := &PropertyValue{}
	pv.PropertyID = Texts("Wikidata")
	pv.Value = Texts(qid)
	return pv
}

// BingEntityID returns an identifier naming a Bing entity, encoded as
// {"propertyID":"BingEntityID","value":"<id>"}.
func BingEntityID(id string) *PropertyValue {
	pv := &PropertyValue{}
	pv.PropertyID = Texts("BingEntityID")
	pv.Value = Texts(id)
	return pv
}

// Entity describes a real-world thing by every identity source known for
// it. Thing and PerSource turn it into nodes for about, mentions,
// knowsAbout, areaServed and similar properties.
type Entity struct {
	Type string // @type; Thing when empty
	ID   string // @id; often a Wikipedia URL
	Name string

	Wikipedia       string   // article title
	Wikidata        string   // item ID, e.g. Q180711
	KGMID           string   // Knowledge Graph machine ID, e.g. /m/019qb_
	KGMIDs          []string // more Knowledge Graph IDs, after KGMID
	DBpedia         string   // resource title
	ProductOntology string   // term
	SameAs          []string
}

// SameAsURLs returns every identity URL of e. The KG machine ID appears as
// a pinned Google search URL.
func (e Entity) SameAsURLs() Values {
	var out Values
	if e.Wikipedia != "" {
		out = append(out, Wikipedia(e.Wikipedia))
	}
	if e.Wikidata != "" {
		out = append(out, Wikidata(e.Wikidata))
	}
	for _, mid := range e.kgmids() {
		out = append(out, KGSearch(mid))
	}
	if e.DBpedia != "" {
		out = append(out, DBpedia(e.DBpedia))
	}
	if e.ProductOntology != "" {
		out = append(out, ProductOntology(e.ProductOntology))
	}
	for _, s := range e.SameAs {
		out = append(out, URL(s))
	}
	return out
}

// Thing returns e as one node carrying every sameAs URL, plus a
// GoogleMachineID identifier for each known KG machine ID.
func (e Entity) Thing() *Thing {
	t := e.base()
	t.SameAs = e.SameAsURLs()
	for _, mid := range e.kgmids() {
		t.Identifier = append(t.Identifier, GoogleMachineID(mid))
	}
	return t
}

// kgmids returns KGMID, when set, followed by KGMIDs.
func (e Entity) kgmids() []string {
	var out []string
	if e.KGMID != "" {
		out = append(out, e.KGMID)
	}
	return append(out, e.KGMIDs...)
}

// PerSource returns one node per sameAs URL, each with the same name, so
// the entity is listed once for every source.
func (e Entity) PerSource() Values {
	var out Values
	for _, u := range e.SameAsURLs() {
		t := e.base()
		t.SameAs = Vals(u)
		out = append(out, t)
	}
	return out
}

func (e Entity) base() *Thing {
	t := &Thing{}
	if e.Type != "" {
		t.Type = Types{e.Type}
	}
	t.ID = e.ID
	if e.Name != "" {
		t.Name = Texts(e.Name)
	}
	return t
}

// Entities returns the Thing node of each entity.
func Entities(es ...Entity) Values {
	out := make(Values, len(es))
	for i, e := range es {
		out[i] = e.Thing()
	}
	return out
}

func underscore(s string) string { return strings.ReplaceAll(s, " ", "_") }
