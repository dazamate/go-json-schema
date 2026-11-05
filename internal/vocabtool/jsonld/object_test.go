package jsonld

import "testing"

// TestDecodeKeepsOrder checks that Decode and Encode keep the key order.
func TestDecodeKeepsOrder(t *testing.T) {
	v, err := Decode([]byte(`{"b":1,"a":2}`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Encode(v, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `{"b":1,"a":2}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestEncodeNoHTMLEscape checks that Encode leaves HTML characters alone.
func TestEncodeNoHTMLEscape(t *testing.T) {
	out, err := Encode("<b>", false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `"<b>"`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
