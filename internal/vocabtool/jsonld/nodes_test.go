package jsonld

import (
	"slices"
	"testing"
)

// TestNodesPaths checks that Nodes lists every object, outermost first.
func TestNodesPaths(t *testing.T) {
	v, err := Decode([]byte(`{"@graph":[{"x":{"y":{}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range Nodes("$", v) {
		got = append(got, n.Path)
	}
	want := []string{"$", "$.@graph[0]", "$.@graph[0].x", "$.@graph[0].x.y"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
