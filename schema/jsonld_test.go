package schema

import (
	"strings"
	"testing"
	"time"
)

func TestJSONLDAddsContextToSingleNode(t *testing.T) {
	n := testNode{}
	n.Name = Texts("seo")
	b, err := JSONLD(n)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"@context":"https://schema.org","@type":"Thing","name":"seo"}`
	if string(b) != want {
		t.Fatalf("got  %s\nwant %s", b, want)
	}
}

func TestGraphWrapsNodesWithContext(t *testing.T) {
	a := testNode{Node: Node{ID: "#a"}}
	g := NewGraph(a, Ref{ID: "#b"})
	b, err := JSONLD(g)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"@context":"https://schema.org","@graph":[{"@type":"Thing","@id":"#a"},{"@id":"#b"}]}`
	if string(b) != want {
		t.Fatalf("got  %s\nwant %s", b, want)
	}
}

func TestGraphWithOneNodeStillEncodesArray(t *testing.T) {
	b, err := JSONLD(NewGraph(Ref{ID: "#a"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"@graph":[{"@id":"#a"}]`) {
		t.Fatalf("got %s", b)
	}
}

func TestScriptTagEscapesClosingTag(t *testing.T) {
	n := testNode{}
	n.Name = Texts("</script><script>alert(1)</script>")
	s, err := ScriptTag(n)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s, `<script type="application/ld+json">`) || !strings.HasSuffix(s, `</script>`) {
		t.Fatalf("bad wrapper: %s", s)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(s, `<script type="application/ld+json">`), `</script>`)
	if strings.Contains(body, "<") {
		t.Fatalf("unescaped < in body: %s", body)
	}
}

func TestDurationOfFormatsISO8601(t *testing.T) {
	cases := map[time.Duration]Duration{
		0:                              "PT0S",
		90 * time.Minute:               "PT1H30M",
		2*time.Minute + 15*time.Second: "PT2M15S",
		26 * time.Hour:                 "PT26H",
	}
	for in, want := range cases {
		if got := DurationOf(in); got != want {
			t.Errorf("DurationOf(%v) = %s, want %s", in, got, want)
		}
	}
}

func TestDateHelpers(t *testing.T) {
	ts := time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC)
	if got := DateOf(ts); got != "2026-10-23" {
		t.Errorf("DateOf = %s", got)
	}
	if got := DateTimeOf(ts); got != "2026-10-23T09:30:00Z" {
		t.Errorf("DateTimeOf = %s", got)
	}
}

func TestEnumMemberEncodesAsSchemaURL(t *testing.T) {
	got := mustMarshal(t, Vals(EnumMember("VeganDiet")))
	if got != `"https://schema.org/VeganDiet"` {
		t.Fatalf("got %s", got)
	}
}

func TestNumbersAndBooleans(t *testing.T) {
	got := mustMarshal(t, Vals(Number(33.6117), Integer(5), Boolean(true)))
	if got != `[33.6117,5,true]` {
		t.Fatalf("got %s", got)
	}
}

func TestRawPassesThrough(t *testing.T) {
	got := mustMarshal(t, Vals(Raw(`{"@list":["Mix","Bake"]}`)))
	if got != `{"@list":["Mix","Bake"]}` {
		t.Fatalf("got %s", got)
	}
}
