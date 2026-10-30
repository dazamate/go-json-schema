package research

import (
	"encoding/json"
	"maps"
	"os"
	"testing"
)

// TestCalibrationTypes types the recorded calibration selections: in the
// moving-cost page only real instances could be typed, and none is (moving
// company is a kind of business, so it stays a Thing); in the forex page the
// trading occupations are Occupations. Everything else stays a Thing.
func TestCalibrationTypes(t *testing.T) {
	typed := func(rep SelectReport) map[string]string {
		out := map[string]string{}
		for _, s := range append(rep.About, rep.Mentions...) {
			var n struct {
				Type string `json:"@type"`
			}
			if err := json.Unmarshal(s.Thing, &n); err != nil {
				t.Fatalf("%s: %v", s.Name, err)
			}
			if n.Type != orElse(s.SchemaType, "Thing") {
				t.Errorf("%s: thing @type %q, schema_type %q", s.Name, n.Type, s.SchemaType)
			}
			if n.Type != "Thing" {
				out[s.Name] = n.Type
			}
		}
		return out
	}

	w := loadWikiFixture(t, "select")
	rep := runSelect(t, calibrationResearcher(t, fakeBrowser{selectPage(w, calibrationPage(t, calibrationURL), nil)}), calibrationQuery())
	if got := typed(rep); len(got) != 0 {
		t.Errorf("moving-cost page: typed %v, want every entity a Thing", got)
	}

	if _, err := os.Stat(forexFixturePath("wiki.json")); err != nil {
		return
	}
	w = loadWikiFixture(t, forexDir)
	var raw rawPage
	if err := json.Unmarshal(readForexFixture(t, "page.json"), &raw); err != nil {
		t.Fatal(err)
	}
	rep = runSelect(t, forexResearcher(t, selectPage(w, raw, nil)), forexQuery())
	want := map[string]string{"Trader (finance)": "Occupation", "Floor trader": "Occupation", "Broker": "Occupation"}
	if got := typed(rep); !maps.Equal(got, want) {
		t.Errorf("forex page: typed %v, want %v", got, want)
	}
}
