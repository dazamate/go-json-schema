package onpage_test

import (
	"reflect"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

// elements is what Elements finds for the draft and the page.
func elements(t *testing.T, draft string, in audit.Input) []onpage.PageElement {
	t.Helper()
	v, err := vocab.Load()
	if err != nil {
		t.Fatal(err)
	}
	c, err := audit.NewContext(v, []byte(draft), in)
	if err != nil {
		t.Fatal(err)
	}
	return onpage.Elements(c)
}

func TestElements(t *testing.T) {
	const draft = `{"@context":"https://schema.org","@graph":[
		{"@type":"WebPage","@id":"https://x.test/#webpage","hasPart":[
			{"@type":"WPHeader","cssSelector":"#top"},
			{"@type":"SiteNavigationElement","cssSelector":["#site-nav","footer nav"]},
			{"@type":"WPFooter"},
			{"@type":"Table","cssSelector":"#nothing"},
			{"@type":"WPSideBar","cssSelector":"#site-nav"}]},
		{"@type":"BreadcrumbList","itemListElement":[]}]}`
	want := []onpage.PageElement{
		{Type: "WPHeader", Selector: "#top", MarkedBy: "$.@graph[0].hasPart[0]"},
		{Type: "SiteNavigationElement", Selector: "#site-nav", Name: "Main", MarkedBy: "$.@graph[0].hasPart[1]"},
		{Type: "BreadcrumbList", Selector: `nav[aria-label="Breadcrumb"]`, Name: "Breadcrumb", MarkedBy: "$.@graph[1]"},
		{Type: "Table", Selector: "body > main > article > table:nth-of-type(1)"},
		{Type: "WPSideBar", Selector: "body > main > article > aside", Name: "Related guides"},
		{Type: "WPAdBlock", Selector: "body > main > article > ins"},
		{Type: "WPFooter", Selector: "body > footer", MarkedBy: "$.@graph[0].hasPart[2]"},
		{Type: "SiteNavigationElement", Selector: "body > footer > nav:nth-of-type(1)", MarkedBy: "$.@graph[0].hasPart[1]"},
		{Type: "SiteNavigationElement", Selector: "body > footer > nav:nth-of-type(2)", MarkedBy: "$.@graph[0].hasPart[1]"},
	}
	if got := elements(t, draft, audit.Input{PageHTML: elementsPage}); !reflect.DeepEqual(got, want) {
		t.Errorf("elements\n got %+v\nwant %+v", got, want)
	}
	if got := elements(t, draft, audit.Input{PageText: "Tea ceremony."}); got != nil {
		t.Errorf("elements without page_html: %+v", got)
	}
	if got := elements(t, draft, audit.Input{}); got != nil {
		t.Errorf("elements without a page: %+v", got)
	}
}

// A page search engines skip still lists its sections, but nothing marks them
// and nothing is asked of the markup.
func TestElementsOnANoindexPage(t *testing.T) {
	const draft = `{"@type":"WebPage","hasPart":[{"@type":"WPFooter","cssSelector":"footer"}]}`
	page := audittest.Noindex(footerPage)
	got := elements(t, draft, audit.Input{PageHTML: page})
	want := []onpage.PageElement{{Type: "WPFooter", Selector: "body > footer"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("elements\n got %+v\nwant %+v", got, want)
	}
	for _, c := range []audit.Check{onpage.InvalidSelector{}, onpage.SelectorWrongType{}, onpage.SelectorMatchesNothing{}, onpage.BareElement{}, onpage.UnmarkedSection{}, onpage.UnmarkedBreadcrumb{}} {
		audittest.Expect(t, c, audittest.Input{Draft: `{"@type":"WebPage"}`, PageHTML: page}, nil)
	}
}
