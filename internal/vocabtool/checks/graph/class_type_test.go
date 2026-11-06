package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

// wikidata is an additionalType of the Wikidata item id.
func wikidata(id string) string { return `"additionalType":"https://www.wikidata.org/wiki/` + id + `"` }

func TestClassType(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"classes that give subtypes", `{"@context":"https://schema.org","@graph":[
			{"@type":"Place","name":"Sydney",` + wikidata("Q515") + `},
			{"@type":"AdministrativeArea","name":"Sydney",` + wikidata("Q515") + `},
			{"@type":"Organization","name":"University of Sydney",` + wikidata("Q3918") + `},
			{"@type":"City","name":"Sydney",` + wikidata("Q515") + `},
			{"@type":"Place","name":"East Asia",` + wikidata("Q82794") + `},
			{"@type":"Place","name":"Blue Mountains","additionalType":["https://www.wikidata.org/wiki/Q46831","https://www.wikidata.org/wiki/Q82794"]}]}`, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: "Place with the Wikidata class Q515: type it City, unless the item is a kind of thing rather than one"},
			{Path: "$.@graph[1]", Severity: audit.Warning, Message: "AdministrativeArea with the Wikidata class Q515: type it City, unless the item is a kind of thing rather than one"},
			{Path: "$.@graph[2]", Severity: audit.Warning, Message: "Organization with the Wikidata class Q3918: type it CollegeOrUniversity, unless the item is a kind of thing rather than one"},
		}},
		{"a landmark", `{"@context":"https://schema.org","@type":"Thing","name":"Alamo Mission","additionalType":["https://www.wikidata.org/wiki/Q1564373"]}`, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: "Thing with the Wikidata class Q1564373: type it LandmarksOrHistoricalBuildings, with TouristAttraction too when visitors go to see it, unless the item is a kind of thing rather than one"},
		}},
		// an item that is itself a class in the table is a kind of thing
		{"a kind of thing", `{"@context":"https://schema.org","@graph":[
			{"@type":"Thing","name":"Moving company","additionalType":["https://www.wikidata.org/wiki/Q783794","https://www.wikidata.org/wiki/Q2169973"],"sameAs":"https://www.wikidata.org/wiki/Q1757263"},
			{"@type":"Thing","name":"Moving company",` + wikidata("Q783794") + `,"identifier":{"@type":"PropertyValue","propertyID":"Wikidata","value":"Q1757263"}},
			{"@type":"Thing","name":"Harbour Removals",` + wikidata("Q783794") + `}]}`, []audit.Issue{
			{Path: "$.@graph[2]", Severity: audit.Warning, Message: "Thing with the Wikidata class Q783794: type it Organization, unless the item is a kind of thing rather than one"},
		}},
		{"no Wikidata class", `{"@type":"Place","name":"Sydney","additionalType":"https://schema.org/City"}`, nil},
		{"two types", `{"@type":["Place","TouristAttraction"],"name":"Sydney",` + wikidata("Q515") + `}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, graph.ClassType{}, audittest.Input{Draft: tc.draft}, tc.want) })
	}
}
