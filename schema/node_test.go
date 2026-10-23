package schema

import (
	"strings"
	"testing"
)

// testProps and testNode stand in for generated blocks so the runtime is
// tested without depending on generated code.
type testProps struct {
	Name   Values `json:"name,omitempty"`
	SameAs Values `json:"sameAs,omitempty"`
}

type testNode struct {
	Node
	testProps
}

func (v testNode) MarshalJSON() ([]byte, error) {
	type plain testNode
	v.Type = v.Type.OrDefault("Thing")
	return EncodeNode(plain(v), v.Extra)
}

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(b)
}

func TestValuesSingleEncodesAsScalar(t *testing.T) {
	got := mustMarshal(t, Texts("Lemon herb chicken"))
	if got != `"Lemon herb chicken"` {
		t.Fatalf("got %s", got)
	}
}

func TestValuesManyEncodesAsArray(t *testing.T) {
	got := mustMarshal(t, URLs("https://a.example", "https://b.example"))
	if got != `["https://a.example","https://b.example"]` {
		t.Fatalf("got %s", got)
	}
}

func TestValuesMixKindsInOneProperty(t *testing.T) {
	got := mustMarshal(t, Vals(Text("Fountain Hills"), URL("http://dbpedia.org/page/Fountain_Hills,_Arizona"), Ref{ID: "https://en.wikipedia.org/wiki/Arizona", Type: Types{"State"}}))
	want := `["Fountain Hills","http://dbpedia.org/page/Fountain_Hills,_Arizona",{"@type":"State","@id":"https://en.wikipedia.org/wiki/Arizona"}]`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestMarshalDoesNotHTMLEscape(t *testing.T) {
	got := mustMarshal(t, URLs("https://www.google.com/search?q=seo&kponly&kgmid=/m/019qb_"))
	if strings.Contains(got, `\u0026`) {
		t.Fatalf("ampersand escaped: %s", got)
	}
}

func TestNodeDefaultTypeAndID(t *testing.T) {
	n := testNode{Node: Node{ID: "https://example.com/#thing"}}
	n.Name = Texts("seo")
	got := mustMarshal(t, n)
	want := `{"@type":"Thing","@id":"https://example.com/#thing","name":"seo"}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestNodeExplicitTypesOverrideDefault(t *testing.T) {
	n := testNode{Node: Node{Type: Types{"Recipe", "Product"}}}
	got := mustMarshal(t, n)
	if got != `{"@type":["Recipe","Product"]}` {
		t.Fatalf("got %s", got)
	}
}

func TestNodeOmitsEmptyProperties(t *testing.T) {
	got := mustMarshal(t, testNode{})
	if got != `{"@type":"Thing"}` {
		t.Fatalf("got %s", got)
	}
}

func TestExtraAddsPropertiesTheStructLacks(t *testing.T) {
	n := testNode{}
	n.Extra = map[string]Values{
		"query-input":   Texts("required name=search_term_string"),
		"addressRegion": Texts("AZ"),
	}
	got := mustMarshal(t, n)
	want := `{"@type":"Thing","addressRegion":"AZ","query-input":"required name=search_term_string"}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestExtraMergesIntoExistingProperty(t *testing.T) {
	n := testNode{}
	n.SameAs = URLs("https://en.wikipedia.org/wiki/Search_engine_optimization")
	n.Extra = map[string]Values{"sameAs": URLs("https://www.wikidata.org/wiki/Q180711", "https://dbpedia.org/resource/Search_engine_optimization")}
	got := mustMarshal(t, n)
	want := `{"@type":"Thing","sameAs":["https://en.wikipedia.org/wiki/Search_engine_optimization","https://www.wikidata.org/wiki/Q180711","https://dbpedia.org/resource/Search_engine_optimization"]}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if strings.Count(got, `"sameAs"`) != 1 {
		t.Fatalf("duplicate key: %s", got)
	}
}

func TestNodeNestsAsValue(t *testing.T) {
	inner := &testNode{}
	inner.Name = Texts("web design")
	outer := testNode{}
	outer.SameAs = Vals(inner)
	got := mustMarshal(t, outer)
	want := `{"@type":"Thing","sameAs":{"@type":"Thing","name":"web design"}}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

type untypedNode struct {
	Node
	testProps
}

func (v untypedNode) MarshalJSON() ([]byte, error) {
	type plain untypedNode
	return EncodeNode(plain(v), v.Extra)
}

func TestEncodeNodeRejectsMissingType(t *testing.T) {
	n := untypedNode{}
	n.Name = Texts("forgot the type")
	if _, err := Marshal(n); err == nil {
		t.Fatal("encoded a node with no @type")
	}
}

func TestTypesOrDefault(t *testing.T) {
	if got := (Types(nil)).OrDefault("Recipe"); len(got) != 1 || got[0] != "Recipe" {
		t.Fatalf("got %v", got)
	}
	if got := (Types{"Recipe", "Product"}).OrDefault("Thing"); len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestRefWithoutIDFails(t *testing.T) {
	if _, err := Marshal(Vals(Ref{Type: Types{"Person"}})); err == nil {
		t.Fatal("encoded a Ref with an empty @id")
	}
}
