package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestPersonName(t *testing.T) {
	const draft = `{"@context":"https://schema.org","@type":"Article","headline":"Tea","author":{"@type":"Person","name":"Ann Lee"}}`
	fires := []audit.Issue{{Path: "$.author", Severity: audit.Warning, Message: `this Person is named "Ann Lee": set givenName Ann and familyName Lee`}}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"a given and a family name", draft, audittest.Input{PageHTML: teaPage}, fires},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, fires},
		// the markup's own faults are held to a page search engines skip too
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"a middle name", `{"@type":"Person","name":"Ann Marie Lee"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `this Person is named "Ann Marie Lee": set givenName Ann and familyName Lee`},
		}},
		{"an honorific", `{"@type":"Person","name":"Dr Jane Smith"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `this Person is named "Dr Jane Smith": set givenName Jane and familyName Smith`},
		}},
		{"an honorific with a stop", `{"@type":"Person","name":"Prof. Jane Smith"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `this Person is named "Prof. Jane Smith": set givenName Jane and familyName Smith`},
		}},
		{"a suffix", `{"@type":"Person","name":"John Smith Jr"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `this Person is named "John Smith Jr": set givenName John and familyName Smith`},
		}},
		{"a degree after a comma", `{"@type":"Person","name":"Jane Smith, PhD"}`, audittest.Input{PageText: "Tea."}, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `this Person is named "Jane Smith, PhD": set givenName Jane and familyName Smith`},
		}},
		{"an honorific and a family name", `{"@type":"Person","name":"Dr Smith"}`, audittest.Input{PageText: "Tea."}, nil},
		{"givenName set", `{"@type":"Person","name":"Ann Lee","givenName":"Ann"}`, audittest.Input{PageText: "Tea."}, nil},
		{"familyName set", `{"@type":"Person","name":"Ann Lee","familyName":"Lee"}`, audittest.Input{PageText: "Tea."}, nil},
		{"one word", `{"@type":"Person","name":"Ann"}`, audittest.Input{PageText: "Tea."}, nil},
		{"not capitalised", `{"@type":"Person","name":"ann lee"}`, audittest.Input{PageText: "Tea."}, nil},
		{"no page", draft, audittest.Input{}, nil},
		// loveitlandscaping.com.au: the business typed Person and
		// Organization, and its posts' author named as the business
		{"a business typed Person too", `{"@context":"https://schema.org","@graph":[
			{"@type":["Person","Organization"],"@id":"https://loveitlandscaping.com.au/#person","name":"Love It Landscaping"},
			{"@type":"WebSite","@id":"https://loveitlandscaping.com.au/#website","name":"Love It Landscaping","publisher":{"@id":"https://loveitlandscaping.com.au/#person"}},
			{"@type":"Person","@id":"https://loveitlandscaping.com.au/author/dyonloveitlandscaping-com-au/","name":"Love It Landscaping"}]}`,
			audittest.Input{PageText: "Landscaping Costs in Melbourne."}, nil},
		// allrecipes.com: the publisher's own test kitchen reviews the recipe
		{"a Person named for the publisher", `{"@context":"https://schema.org","@type":"Recipe","name":"Best Chocolate Chip Cookies",
			"publisher":{"@type":"Organization","name":"Allrecipes"},
			"mainEntityOfPage":{"@type":"WebPage","reviewedBy":{"@type":"Person","name":"Allrecipes Test Kitchen"}},
			"author":{"@type":"Person","name":"Dora"}}`,
			audittest.Input{PageText: "Best Chocolate Chip Cookies."}, nil},
		// jamieoliver.com: a person the site is named after is still a person
		{"a person the site is named after", `{"@context":"https://schema.org","@type":"Recipe","name":"Easy chicken curry",
			"author":{"@type":"Person","name":"Jamie Oliver"}}`,
			audittest.Input{PageText: "Easy chicken curry.", PageURL: "https://www.jamieoliver.com/recipes/chicken/easy-chicken-curry/"}, []audit.Issue{
				{Path: "$.author", Severity: audit.Warning, Message: `this Person is named "Jamie Oliver": set givenName Jamie and familyName Oliver`},
			}},
		// a personal site: the WebSite is named after the person who runs it
		{"a WebSite named after its person", `{"@context":"https://schema.org","@graph":[
			{"@type":"WebSite","@id":"https://janesmith.test/#website","name":"Jane Smith","publisher":{"@id":"https://janesmith.test/#person"}},
			{"@type":"Person","@id":"https://janesmith.test/#person","name":"Jane Smith"}]}`,
			audittest.Input{PageText: "Jane Smith, illustrator."}, []audit.Issue{
				{Path: "$.@graph[1]", Severity: audit.Warning, Message: `this Person is named "Jane Smith": set givenName Jane and familyName Smith`},
			}},
		// Yoast and Rank Math type the site's person Person and Organization
		{"a person typed Organization too", `{"@context":"https://schema.org","@graph":[
			{"@type":["Person","Organization"],"@id":"https://janesmith.test/#person","name":"Jane Smith"}]}`,
			audittest.Input{PageText: "Jane Smith, illustrator."}, []audit.Issue{
				{Path: "$.@graph[0]", Severity: audit.Warning, Message: `this Person is named "Jane Smith": set givenName Jane and familyName Smith`},
			}},
		// a business's name holds words no person's name does
		{"a business name typed Person", `{"@type":"Person","name":"The Bread Kitchen"}`, audittest.Input{PageText: "Sourdough."}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, graph.PersonName{}, tc.in, tc.want)
		})
	}
}
