package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestWebPageElement(t *testing.T) {
	const faq = `FAQPage, as a part of the page (isPartOf the WebPage, in its hasPart) with the Questions as mainEntity`
	generic := func(path, use string) audit.Issue {
		return audit.Issue{Path: path, Severity: audit.Warning, Message: "WebPageElement is generic: use " + use}
	}
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"a section a more specific type fits", `{"@context":"https://schema.org","@graph":[
			{"@type":"WebPageElement","@id":"https://x.test/#faq","mainEntity":[{"@type":"Question","name":"Q?","acceptedAnswer":{"@type":"Answer","text":"A."}}]},
			{"@type":"WebPageElement","name":"Frequently asked questions"},
			{"@type":"WebPageElement","name":"Main navigation"},
			{"@type":"WebPageElement","name":"Site header"},
			{"@type":"WebPageElement","name":"Footer"},
			{"@type":"WebPageElement","name":"Sidebar"},
			{"@type":"WebPageElement","name":"Price table"},
			{"@type":"WebPageElement","name":"Gallery"},
			{"@type":"FAQPage","@id":"https://x.test/#faq2","mainEntity":[{"@type":"Question","name":"Q?","acceptedAnswer":{"@type":"Answer","text":"A."}}]},
			{"@type":"SiteNavigationElement","name":"Main navigation"}]}`, []audit.Issue{
			generic("$.@graph[0]", faq),
			generic("$.@graph[1]", faq),
			generic("$.@graph[2]", "SiteNavigationElement"),
			generic("$.@graph[3]", "WPHeader"),
			generic("$.@graph[4]", "WPFooter"),
			generic("$.@graph[5]", "WPSideBar"),
			generic("$.@graph[6]", "Table"),
		}},
		{"a headline names it", `{"@type":"WebPageElement","headline":"Advertisement"}`, []audit.Issue{generic("$", "WPAdBlock")}},
		{"a name that fits no type", `{"@type":"WebPageElement","name":"Gallery"}`, nil},
		{"a type that is more than a WebPageElement", `{"@type":["WebPageElement","Table"],"name":"Price table"}`, nil},
		// sections named by their headings on real pages: a restaurant's
		// menus (vicisouthbank.com.au), a wedding checklist's steps
		// (firstlookweddings.co.uk), a blog's heading (orbitmedia.com), a
		// list of questions to ask (redfin.com) and a contact prompt
		// (royalcanadiancollective.com) are content, not page chrome
		{"content sections named with chrome words", `{"@context":"https://schema.org","@graph":[
			{"@type":"WebPageElement","name":"Kids Menu"},
			{"@type":"WebPageElement","name":"Our Menus"},
			{"@type":"WebPageElement","name":"Menu"},
			{"@type":"WebPageElement","name":"$69PP WHOLE TABLE ONLY (MIN. 2 PPL)"},
			{"@type":"WebPageElement","name":"Finalise guest list and table plan"},
			{"@type":"WebPageElement","name":"AI is table stakes. It doesn’t give marketers an edge."},
			{"@type":"WebPageElement","name":"Table of contents"},
			{"@type":"WebPageElement","name":"18 Questions to Ask When Touring House on Virtually"},
			{"@type":"WebPageElement","name":"Still have questions?"}]}`, nil},
		// the same sites' chrome and data, named as they name them
		{"chrome and data sections", `{"@context":"https://schema.org","@graph":[
			{"@type":"WebPageElement","name":"Spa Main Menu"},
			{"@type":"WebPageElement","name":"Primary menu"},
			{"@type":"WebPageElement","name":"Site Menu"},
			{"@type":"WebPageElement","name":"Questions & answers"},
			{"@type":"WebPageElement","name":"Nutrition Facts table"},
			{"@type":"WebPageElement","name":"Table of fees"}]}`, []audit.Issue{
			generic("$.@graph[0]", "SiteNavigationElement"),
			generic("$.@graph[1]", "SiteNavigationElement"),
			generic("$.@graph[2]", "SiteNavigationElement"),
			generic("$.@graph[3]", faq),
			generic("$.@graph[4]", "Table"),
			generic("$.@graph[5]", "Table"),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, graph.WebPageElement{}, audittest.Input{Draft: tc.draft}, tc.want)
		})
	}
}
