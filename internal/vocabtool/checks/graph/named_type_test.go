package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestNamedType(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		// a page about weddings, a Thing (typed by its Wikidata class instead),
		// a Wikidata entity, and names no subtype matches get none: News is not
		// stemmed to "new", and House, Country and State are left to other
		// signals
		{"names and URLs that name subtypes", `{"@context":"https://schema.org","@graph":[
			{"@type":"WebPage","@id":"https://x.test/about/#webpage","name":"Andrew Mo"},
			{"@type":"WebPage","url":"https://x.test/contact/","name":"Get in touch"},
			{"@type":"WebPage","name":"Image gallery"},
			{"@type":"LocalBusiness","name":"Park Street Bakery"},
			{"@type":"Organization","name":"University of Sydney"},
			{"@type":"Place","name":"Bondi Beach"},
			{"@type":"WebPage","name":"Wedding MC Sydney","url":"https://x.test/wedding-mc-sydney/"},
			{"@type":"Thing","name":"Wedding customs by country"},
			{"@type":"Place","name":"Hyde Park","sameAs":"https://www.wikidata.org/wiki/Q1"},
			{"@type":"Article","headline":"New wedding traditions for 2027"},
			{"@type":"Place","name":"Sydney Opera House"},
			{"@type":"WebPage","url":"https://x.test/contact/","additionalType":["https://www.wikidata.org/wiki/Q49836","https://schema.org/WebPage"]}]}`, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: `WebPage "Andrew Mo": its name or URL names the subtype AboutPage; type it AboutPage`},
			{Path: "$.@graph[1]", Severity: audit.Warning, Message: `WebPage "Get in touch": its name or URL names the subtype ContactPage; type it ContactPage`},
			{Path: "$.@graph[2]", Severity: audit.Warning, Message: `WebPage "Image gallery": its name or URL names the subtype ImageGallery; type it ImageGallery`},
			{Path: "$.@graph[3]", Severity: audit.Warning, Message: `LocalBusiness "Park Street Bakery": its name or URL names the subtype Bakery; type it Bakery`},
			{Path: "$.@graph[4]", Severity: audit.Warning, Message: `Organization "University of Sydney": its name or URL names the subtype CollegeOrUniversity; type it CollegeOrUniversity`},
			{Path: "$.@graph[5]", Severity: audit.Warning, Message: `Place "Bondi Beach": its name or URL names the subtype Beach; type it Beach`},
			{Path: "$.@graph[11]", Severity: audit.Warning, Message: `WebPage "https://x.test/contact/": its name or URL names the subtype ContactPage; type it ContactPage`},
		}},
		{"two subtypes as deep", `{"@type":"WebPage","name":"About and contact"}`, []audit.Issue{
			{Path: "$", Severity: audit.Warning, Message: `WebPage "About and contact": its name or URL names the subtype AboutPage or ContactPage; type it AboutPage or ContactPage`},
		}},
		// a breadcrumb's items name the pages they link
		{"breadcrumb items", `{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
			{"@type":"ListItem","position":1,"name":"About us","item":"https://x.test/about/"},
			{"@type":"ListItem","position":2,"name":"How to get the cheapest long distance moving quote","item":"https://x.test/how-to-get-a-quote/"}]}`, nil},
		// a property only its subtypes take already told this node
		{"a node another signal told", `{"@type":"Organization","name":"University of Sydney","openingHours":"Mo-Fr 09:00-17:00"}`, nil},
		{"a node with a Wikidata class", `{"@type":"Organization","name":"University of Sydney",` + wikidata("Q3918") + `}`, nil},
		{"a WebPageElement", `{"@type":"WebPageElement","name":"Site navigation"}`, nil},
		// headings and titles from real pages: a type word that only
		// qualifies another word, or sits in a place's name or a
		// prepositional phrase, names no subtype
		{"a type word that is not the head", `{"@context":"https://schema.org","@graph":[
			{"@type":"LocalBusiness","name":"Enjoy Italian Food at Vici South Bank | Pizza & Pasta Brisbane"},
			{"@type":"Article","headline":"LIM Reports NZ: What Property Buyers Need to Know"},
			{"@type":"WebPage","name":"Find out about legal and business responsibilities in your industry."},
			{"@type":"Service","name":"Payment Offers"},
			{"@type":"WebPage","name":"How to Choose the Right Real Estate Agent in 10 Steps","url":"https://www.redfin.com/blog/how-to-choose-a-real-estate-agent/"}]}`, nil},
		// the same pages' names whose head is the type
		{"a type word that is the head", `{"@context":"https://schema.org","@graph":[
			{"@type":"LocalBusiness","name":"Opus Vancouver Boutique Hotel in Yaletown"},
			{"@type":"LocalBusiness","name":"The Willows Beauty Salon"},
			{"@type":"Service","name":"Personal loans"},
			{"@type":"WebPage","name":"Pricing Page FAQs"},
			{"@type":"WebPage","name":"Frequently Asked Questions","url":"https://royalcanadiancollective.com/pages/faq"}]}`, []audit.Issue{
			{Path: "$.@graph[0]", Severity: audit.Warning, Message: `LocalBusiness "Opus Vancouver Boutique Hotel in Yaletown": its name or URL names the subtype Hotel; type it Hotel`},
			{Path: "$.@graph[1]", Severity: audit.Warning, Message: `LocalBusiness "The Willows Beauty Salon": its name or URL names the subtype BeautySalon; type it BeautySalon`},
			{Path: "$.@graph[2]", Severity: audit.Warning, Message: `Service "Personal loans": its name or URL names the subtype LoanOrCredit; type it LoanOrCredit`},
			{Path: "$.@graph[3]", Severity: audit.Warning, Message: `WebPage "Pricing Page FAQs": its name or URL names the subtype FAQPage; type it FAQPage`},
			{Path: "$.@graph[4]", Severity: audit.Warning, Message: `WebPage "Frequently Asked Questions": its name or URL names the subtype FAQPage; type it FAQPage`},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, graph.NamedType{}, audittest.Input{Draft: tc.draft}, tc.want) })
	}
}
