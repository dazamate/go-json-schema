package vocabtool

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func newTool(t *testing.T) *Tool {
	t.Helper()
	tool, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return tool
}

func TestTypePropertiesListsInheritedProperties(t *testing.T) {
	info, err := newTool(t).TypeProperties("Recipe")
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Properties) != 148 {
		t.Fatalf("Recipe has %d properties, want 148", len(info.Properties))
	}
	byName := map[string]PropertyRef{}
	for _, p := range info.Properties {
		byName[p.Name] = p
	}
	if p := byName["recipeIngredient"]; p.From != "Recipe" || !slices.Contains(p.Expects, "Text") {
		t.Errorf("recipeIngredient = %+v", p)
	}
	if p := byName["sameAs"]; p.From != "Thing" {
		t.Errorf("sameAs from %q, want Thing", p.From)
	}
	if info.GoType != "Recipe" || !slices.Contains(info.Ancestors, "CreativeWork") {
		t.Errorf("info = %+v", info)
	}
}

func TestTypePropertiesSuggestsCasing(t *testing.T) {
	_, err := newTool(t).TypeProperties("localbusiness")
	if err == nil || !strings.Contains(err.Error(), "LocalBusiness") {
		t.Fatalf("err = %v, want a LocalBusiness suggestion", err)
	}
}

func TestEnumValuesIncludeSubEnumerationMembers(t *testing.T) {
	info, err := newTool(t).EnumValues("ItemAvailability")
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Members) != 12 || info.Members[0].URL != "https://schema.org/BackOrder" {
		t.Fatalf("members = %+v", info.Members)
	}
	spec, err := newTool(t).EnumValues("Specialty")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(spec.Members, func(m MemberRef) bool { return m.Name == "Radiography" }) {
		t.Error("Specialty lacks Radiography (a MedicalSpecialty member)")
	}
}

func TestPropertyInfoNamesGoBlock(t *testing.T) {
	tool := newTool(t)
	info, err := tool.PropertyInfo("knowsAbout")
	if err != nil {
		t.Fatal(err)
	}
	if info.GoField != "KnowsAbout" || info.GoBlock != "KnowsAboutTrait" || len(info.Domains) < 2 {
		t.Errorf("knowsAbout = %+v", info)
	}
	avail, err := tool.PropertyInfo("availability")
	if err != nil {
		t.Fatal(err)
	}
	if avail.Enum != "ItemAvailability" || !slices.Contains(avail.Members, "InStock") {
		t.Errorf("availability = %+v", avail)
	}
	if ing, _ := tool.PropertyInfo("recipeIngredient"); ing.GoBlock != "RecipeProps" {
		t.Errorf("recipeIngredient block = %q", ing.GoBlock)
	}
}

func check(t *testing.T, draft string) Report {
	t.Helper()
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return r
}

func issueAt(r Report, severity, path, contains string) bool {
	for _, i := range r.Issues {
		if i.Severity == severity && i.Path == path && strings.Contains(i.Message, contains) {
			return true
		}
	}
	return false
}

func TestCheckFlagsUnknownTypesAndProperties(t *testing.T) {
	r := check(t, `{"@type":"Recipee","name":"x"}`)
	if !issueAt(r, Error, "$", `unknown type "Recipee"`) || r.Errors != 1 {
		t.Fatalf("issues = %+v", r.Issues)
	}
	r = check(t, `{"@type":"Recipe","recipeingredient":["1 lemon"]}`)
	if !issueAt(r, Error, "$.recipeingredient", "recipeIngredient") {
		t.Fatalf("issues = %+v", r.Issues)
	}
}

func TestCheckKeepsPropertyOutsideItsDomainAsInfo(t *testing.T) {
	r := check(t, `{"@type":"Recipe","knowsAbout":"Lemon"}`)
	if r.Errors != 0 || !issueAt(r, Info, "$.knowsAbout", "not listed for Recipe") {
		t.Fatalf("issues = %+v", r.Issues)
	}
	if !strings.Contains(r.ScriptTag, `"knowsAbout":"Lemon"`) {
		t.Fatalf("knowsAbout dropped: %s", r.ScriptTag)
	}
}

func TestCheckNormalizesAndValidatesEnumMembers(t *testing.T) {
	r := check(t, `{"@type":"Offer","availability":"InStock","itemCondition":"https://schema.org/NewCondition"}`)
	if r.Errors != 0 || !strings.Contains(r.ScriptTag, `"availability":"https://schema.org/InStock"`) {
		t.Fatalf("issues = %+v\n%s", r.Issues, r.ScriptTag)
	}
	r = check(t, `{"@type":"Offer","availability":"InStok"}`)
	if !issueAt(r, Error, "$.availability", "not an ItemAvailability member") {
		t.Fatalf("issues = %+v", r.Issues)
	}
}

func TestCheckWalksNestedNodesAndGraphs(t *testing.T) {
	r := check(t, `[{"@type":"Recipe","offers":{"@type":"Offer","availability":"Nope"}},{"@type":"Persn"}]`)
	if !issueAt(r, Error, "$.@graph[0].offers.availability", "ItemAvailability") || !issueAt(r, Error, "$.@graph[1]", `unknown type "Persn"`) {
		t.Fatalf("issues = %+v", r.Issues)
	}
	if !strings.Contains(r.ScriptTag, `{"@context":"https://schema.org","@graph":[`) {
		t.Fatalf("array not wrapped in a graph: %s", r.ScriptTag)
	}
}

func TestCheckAcceptsPrefixedTermsAndReferences(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@type":["schema:Recipe","https://schema.org/HowTo"],"schema:name":"x","author":{"@id":"#me"}}`)
	if len(r.Issues) != 0 {
		t.Fatalf("issues = %+v", r.Issues)
	}
}

func TestCheckWarnsOnUntypedNodeWithProperties(t *testing.T) {
	r := check(t, `{"@type":"Recipe","author":{"name":"Jane"}}`)
	if !issueAt(r, Warning, "$.author", "no @type") {
		t.Fatalf("issues = %+v", r.Issues)
	}
}

func TestCheckKeepsKeyOrderAndEscapesScript(t *testing.T) {
	r := check(t, `{"@type":"Recipe","name":"</script><b>","description":"d","about":[]}`)
	want := `<script type="application/ld+json">{"@context":"https://schema.org","@type":"Recipe","name":"</script><b>","description":"d","about":[]}</script>`
	if r.ScriptTag != want {
		t.Fatalf("got  %s\nwant %s", r.ScriptTag, want)
	}
}

func TestSuggestPicksClosestTerm(t *testing.T) {
	tool := newTool(t)
	for in, want := range map[string]string{"Persn": "Person", "Recipee": "Recipe", "recipeingredient": "recipeIngredient"} {
		kind := kindType
		if in == "recipeingredient" {
			kind = kindProperty
		}
		if got := tool.suggest(kind, in); got != want {
			t.Errorf("suggest(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckWarnsOnARepeatedWikidataItem(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"City","@id":"https://x.test/#sydney","name":"Sydney","sameAs":["https://www.wikidata.org/wiki/Q3130"]},
		{"@type":"WebPage","@id":"https://x.test/#webpage",
		 "about":[{"@type":"City","name":"Sydney","sameAs":["https://en.wikipedia.org/wiki/Sydney","https://www.wikidata.org/wiki/Q3130"]},
		          {"@id":"https://x.test/#sydney"},
		          {"@type":"Thing","name":"Wedding","sameAs":"https://www.wikidata.org/wiki/Q49836"}]}]}`)
	if !issueAt(r, Warning, "$.@graph[1].about[0]", `put {"@id": "https://x.test/#sydney"} here`) {
		t.Errorf("no repeat warning on the inline Sydney: %+v", r.Issues)
	}
	n := 0
	for _, i := range r.Issues {
		if strings.Contains(i.Message, "same Wikidata item") {
			n++
		}
	}
	if n != 1 || r.Errors != 0 {
		t.Errorf("%d repeat warnings, %d errors, want 1 and 0: %+v", n, r.Errors, r.Issues)
	}
}

// issueCount counts the issues whose message contains the text.
func issueCount(r Report, contains string) int {
	n := 0
	for _, i := range r.Issues {
		if strings.Contains(i.Message, contains) {
			n++
		}
	}
	return n
}

func TestCheckWarnsOnAThingWithATypedClass(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","about":[
		{"@type":"Thing","name":"Sydney","additionalType":["https://www.wikidata.org/wiki/Q515","https://www.wikidata.org/wiki/Q200250"]},
		{"@type":"Thing","name":"Mandarin Chinese","additionalType":"https://www.wikidata.org/wiki/Q34770"},
		{"@type":"Thing","name":"Varieties of Chinese","additionalType":["https://www.wikidata.org/wiki/Q7850","https://www.wikidata.org/wiki/Q33384"]},
		{"@type":"Thing","name":"Wedding","additionalType":"https://www.wikidata.org/wiki/Q2245405"},
		{"@type":"Thing","name":"Tea"},
		{"@type":"City","name":"Sydney","additionalType":"https://www.wikidata.org/wiki/Q515"}]}]}`)
	if !issueAt(r, Warning, "$.@graph[0].about[0]", "type it City") {
		t.Errorf("no City warning on the Sydney Thing: %+v", r.Issues)
	}
	if !issueAt(r, Warning, "$.@graph[0].about[1]", "type it Language") {
		t.Errorf("no Language warning on the Mandarin Thing: %+v", r.Issues)
	}
	// Only the first class counts: it is an instance-of class, while a second
	// may be a subclass-of one (Varieties of Chinese is a subclass of dialect).
	if n := issueCount(r, "type it"); n != 2 || r.Errors != 0 {
		t.Errorf("%d type warnings, %d errors, want 2 and 0: %+v", n, r.Errors, r.Issues)
	}
}

func TestCheckWarnsOnEntitiesThePageDoesNotName(t *testing.T) {
	draft := `{"@context":"https://schema.org","@graph":[
		{"@type":"City","@id":"https://x.test/#sydney","name":"Sydney"},
		{"@type":"Occupation","@id":"https://x.test/#chef","name":"Chef"},
		{"@type":"WebSite","@id":"https://x.test/#website","about":[{"@type":"Thing","name":"Catering"}]},
		{"@type":"Organization","@id":"https://x.test/#org","knowsAbout":[{"@type":"Thing","name":"Catering"}]},
		{"@type":"WebPage","@id":"https://x.test/#webpage",
		 "about":[{"@type":"Thing","name":"Toast (honor)"},{"@type":"Thing","name":"Bride"},{"@id":"https://x.test/#sydney"},
		          {"@type":"Thing","name":"Groomsman","alternateName":["groomsmen"]},{"@type":"Thing","name":"Hajong marriage"}],
		 "mentions":[{"@type":"Thing","name":"Pre-wedding customs"},{"@id":"https://x.test/#chef"},"Kitchen",{"@type":"Thing","name":"Chinese Australians"}]}]}`
	text := "Our Sydney MC gives the toast, guides the brides and the groomsmen through pre wedding customs for Chinese-Australian families."
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{PageText: text})
	if err != nil {
		t.Fatal(err)
	}
	for path, name := range map[string]string{
		"$.@graph[4].about[4]":    "Hajong marriage",
		"$.@graph[4].mentions[1]": "Chef",
		"$.@graph[4].mentions[2]": "Kitchen",
	} {
		if !issueAt(r, Warning, path, name+" is not named on the page") {
			t.Errorf("no not-on-page warning for %s at %s: %+v", name, path, r.Issues)
		}
	}
	// Toast (honor), brides, the groomsmen alias, #sydney by its node's name,
	// "pre wedding" and Chinese-Australian are all on the page; the WebSite's about and
	// knowsAbout are about the whole site, not this page.
	if n := issueCount(r, "is not named on the page"); n != 3 || r.Errors != 0 {
		t.Errorf("%d not-on-page warnings, %d errors, want 3 and 0: %+v", n, r.Errors, r.Issues)
	}
	if n := issueCount(check(t, draft), "is not named on the page"); n != 0 {
		t.Errorf("%d not-on-page warnings without page text, want 0", n)
	}
}

func TestCheckWarnsOnAWebPageElementWithAMoreSpecificType(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"WebPageElement","@id":"https://x.test/#faq","mainEntity":[{"@type":"Question","name":"Q?","acceptedAnswer":{"@type":"Answer","text":"A."}}]},
		{"@type":"WebPageElement","name":"Frequently asked questions"},
		{"@type":"WebPageElement","name":"Main navigation"},
		{"@type":"WebPageElement","name":"Site header"},
		{"@type":"WebPageElement","name":"Footer"},
		{"@type":"WebPageElement","name":"Sidebar"},
		{"@type":"WebPageElement","name":"Price table"},
		{"@type":"WebPageElement","name":"Gallery"},
		{"@type":"FAQPage","@id":"https://x.test/#faq2","mainEntity":[{"@type":"Question","name":"Q?","acceptedAnswer":{"@type":"Answer","text":"A."}}]},
		{"@type":"SiteNavigationElement","name":"Main navigation"}]}`)
	for path, want := range map[string]string{
		"$.@graph[0]": "FAQPage",
		"$.@graph[1]": "FAQPage",
		"$.@graph[2]": "SiteNavigationElement",
		"$.@graph[3]": "WPHeader",
		"$.@graph[4]": "WPFooter",
		"$.@graph[5]": "WPSideBar",
		"$.@graph[6]": "Table",
	} {
		if !issueAt(r, Warning, path, "use "+want) {
			t.Errorf("no %s warning at %s: %+v", want, path, r.Issues)
		}
	}
	if n := issueCount(r, "WebPageElement is generic"); n != 7 {
		t.Errorf("%d WebPageElement warnings, want 7 (none on Gallery or the specific types): %+v", n, r.Issues)
	}
}

// elementsPage has one of each section a specific type describes, plus the
// ones that must be skipped: an article's own header, a presentation table,
// a template, a hidden nav and a script.
const elementsPage = `<!doctype html><html><head><title>Run sheet</title><style>nav{}</style></head><body>
<header id="top"><nav id="site-nav" aria-label="Main"><a href="/">Home</a></nav></header>
<nav aria-label="Breadcrumb"><ol><li>Home</li></ol></nav>
<main><article><header><h1>Banquet run sheet</h1></header>
<table><tr><td>6pm</td><td>Doors open</td></tr></table>
<table role="presentation"><tr><td>layout</td></tr></table>
<aside><h2>Related guides</h2><p>Tea ceremony</p></aside>
<ins class="adsbygoogle" data-ad-slot="1"></ins>
</article></main>
<footer><nav><a href="/contact/">Contact</a></nav><nav><a href="/privacy/">Privacy</a></nav></footer>
<template><nav id="ghost"></nav></template>
<nav hidden id="hidden-nav"></nav>
<script>var x = "<nav>Kitchen</nav>";</script>
</body></html>`

func TestCheckListsThePageElements(t *testing.T) {
	draft := `{"@context":"https://schema.org","@graph":[
		{"@type":"WebPage","@id":"https://x.test/#webpage","about":[{"@type":"Thing","name":"Tea ceremony"},{"@type":"Thing","name":"Kitchen"}],"hasPart":[
			{"@type":"WPHeader","cssSelector":"#top"},
			{"@type":"SiteNavigationElement","cssSelector":["#site-nav","footer nav"]},
			{"@type":"WPFooter"},
			{"@type":"Table","cssSelector":"#nothing"},
			{"@type":"WPSideBar","cssSelector":"#site-nav"}]},
		{"@type":"BreadcrumbList","itemListElement":[]}]}`
	r, err := newTool(t).Check(json.RawMessage(draft), CheckOptions{PageHTML: elementsPage})
	if err != nil {
		t.Fatal(err)
	}
	type want struct{ typ, selector, name, markedBy string }
	wants := []want{
		{"WPHeader", "#top", "", "$.@graph[0].hasPart[0]"},
		{"SiteNavigationElement", "#site-nav", "Main", "$.@graph[0].hasPart[1]"},
		{"BreadcrumbList", `nav[aria-label="Breadcrumb"]`, "Breadcrumb", "$.@graph[1]"},
		{"Table", "body > main > article > table:nth-of-type(1)", "", ""},
		{"WPSideBar", "body > main > article > aside", "Related guides", ""},
		{"WPAdBlock", "body > main > article > ins", "", ""},
		{"WPFooter", "body > footer", "", "$.@graph[0].hasPart[2]"},
		{"SiteNavigationElement", "body > footer > nav:nth-of-type(1)", "", "$.@graph[0].hasPart[1]"},
		{"SiteNavigationElement", "body > footer > nav:nth-of-type(2)", "", "$.@graph[0].hasPart[1]"},
	}
	if len(r.PageElements) != len(wants) {
		t.Fatalf("%d page elements, want %d: %+v", len(r.PageElements), len(wants), r.PageElements)
	}
	for i, w := range wants {
		got := r.PageElements[i]
		if got.Type != w.typ || got.Selector != w.selector || got.Name != w.name || got.MarkedBy != w.markedBy {
			t.Errorf("element %d = %+v, want %+v", i, got, w)
		}
	}
	for path, msg := range map[string]string{
		"html:body > main > article > table:nth-of-type(1)": "has no Table",
		"html:body > main > article > aside":                "has no WPSideBar",
		"html:body > main > article > ins":                  "has no WPAdBlock",
		"$.@graph[0].hasPart[3]":                            `cssSelector "#nothing" matches nothing on the page`,
		"$.@graph[0].hasPart[4]":                            `cssSelector "#site-nav" selects a SiteNavigationElement, not a WPSideBar`,
	} {
		if !issueAt(r, Warning, path, msg) {
			t.Errorf("no warning %q at %s: %+v", msg, path, r.Issues)
		}
	}
	// page_text comes from the HTML when it is not given: the aside's "Tea
	// ceremony" is on the page, the script's "Kitchen" is not.
	if n := issueCount(r, "is not named on the page"); n != 1 || !issueAt(r, Warning, "$.@graph[0].about[1]", "Kitchen is not named") {
		t.Errorf("%d not-on-page warnings, want 1 for Kitchen: %+v", n, r.Issues)
	}
	if n := r.Warnings; n != 6 || r.Errors != 0 {
		t.Errorf("%d warnings, %d errors, want 6 and 0: %+v", n, r.Errors, r.Issues)
	}
	if plain := check(t, draft); len(plain.PageElements) != 0 || issueCount(plain, "has no") != 0 {
		t.Errorf("page elements without page_html: %+v", plain.PageElements)
	}
}

func TestCheckSuggestsTheSubtypeThatTakesAProperty(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"Organization","name":"Acme","openingHours":"Mo-Fr 09:00-17:00"},
		{"@type":"CreativeWork","name":"Notes","articleBody":"Text."},
		{"@type":"Recipe","name":"Tart","knowsAbout":"Lemon"}]}`)
	// both subtypes that take it are named, the most general ones only
	if !issueAt(r, Warning, "$.@graph[0].openingHours", "subtypes EducationalOrganization and LocalBusiness: type it EducationalOrganization or LocalBusiness") {
		t.Errorf("no EducationalOrganization or LocalBusiness warning for openingHours on an Organization: %+v", r.Issues)
	}
	if !issueAt(r, Warning, "$.@graph[1].articleBody", "type it Article") {
		t.Errorf("no Article warning for articleBody on a CreativeWork: %+v", r.Issues)
	}
	// knowsAbout is no subtype of Recipe's: it stays an info, kept as is
	if !issueAt(r, Info, "$.@graph[2].knowsAbout", "not listed for Recipe") || r.Warnings != 2 || r.Errors != 0 {
		t.Errorf("issues = %+v, want two warnings and the knowsAbout info", r.Issues)
	}
}

func TestCheckSuggestsTheSubtypeItsWikidataClassGives(t *testing.T) {
	q := func(id string) string { return `"additionalType":"https://www.wikidata.org/wiki/` + id + `"` }
	r := check(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"Place","name":"Sydney",`+q("Q515")+`},
		{"@type":"AdministrativeArea","name":"Sydney",`+q("Q515")+`},
		{"@type":"Organization","name":"University of Sydney",`+q("Q3918")+`},
		{"@type":"City","name":"Sydney",`+q("Q515")+`},
		{"@type":"Place","name":"East Asia",`+q("Q82794")+`}]}`)
	for path, want := range map[string]string{"$.@graph[0]": "City", "$.@graph[1]": "City", "$.@graph[2]": "CollegeOrUniversity"} {
		if !issueAt(r, Warning, path, "type it "+want) {
			t.Errorf("no %s warning at %s: %+v", want, path, r.Issues)
		}
	}
	if r.Warnings != 3 || r.Errors != 0 {
		t.Errorf("%d warnings, %d errors, want 3 and 0: %+v", r.Warnings, r.Errors, r.Issues)
	}
}

func TestCheckSuggestsTheSubtypeItsNameOrURLNames(t *testing.T) {
	r := check(t, `{"@context":"https://schema.org","@graph":[
		{"@type":"WebPage","@id":"https://x.test/about/#webpage","name":"Andrew Mo"},
		{"@type":"WebPage","url":"https://x.test/contact/","name":"Get in touch"},
		{"@type":"WebPage","name":"Image gallery"},
		{"@type":"LocalBusiness","name":"Park Street Bakery"},
		{"@type":"Organization","name":"University of Sydney"},
		{"@type":"Place","name":"Bondi Beach"},
		{"@type":"WebPage","name":"Wedding MC Sydney","url":"https://x.test/wedding-mc-sydney/"},
		{"@type":"Thing","name":"Wedding customs by country"},
		{"@type":"Place","name":"Hyde Park","sameAs":"https://www.wikidata.org/wiki/Q1"},
		{"@type":"EntertainmentBusiness","name":"Andrew Mo, Sydney Wedding MC"},
		{"@type":"Service","name":"Chinese wedding MC"},
		{"@type":"Article","headline":"New wedding traditions for 2027"},
		{"@type":"Place","name":"Sydney Opera House"},
		{"@type":"Place","name":"Hunter Valley wine country"},
		{"@type":"Place","name":"State Theatre"},
		{"@type":"CreativeWork","name":"How we plan your day"},
		{"@type":"Event","name":"Supply and demand talk"}]}`)
	for path, want := range map[string]string{
		"$.@graph[0]": "AboutPage",
		"$.@graph[1]": "ContactPage",
		"$.@graph[2]": "ImageGallery",
		"$.@graph[3]": "Bakery",
		"$.@graph[4]": "CollegeOrUniversity",
		"$.@graph[5]": "Beach",
	} {
		if !issueAt(r, Warning, path, "type it "+want) {
			t.Errorf("no %s warning at %s: %+v", want, path, r.Issues)
		}
	}
	// a page about weddings, a Thing (typed by its Wikidata class instead),
	// a Wikidata entity, and names no subtype matches get none. News is not
	// stemmed to "new"; House, Country and State are left to other signals,
	// since venues, regions and theatres carry those words in their names;
	// HowTo needs "how" and "to", OnDemandEvent "on" and "demand".
	if r.Warnings != 6 || r.Errors != 0 {
		t.Errorf("%d warnings, %d errors, want 6 and 0: %+v", r.Warnings, r.Errors, r.Issues)
	}
}
