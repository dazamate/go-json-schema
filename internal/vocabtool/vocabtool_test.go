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
