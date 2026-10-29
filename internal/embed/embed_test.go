package embed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

func words(n int) string {
	w := make([]string, n)
	for i := range w {
		w[i] = fmt.Sprintf("w%d", i)
	}
	return strings.Join(w, " ")
}

func TestWindowsMatchesTheReference(t *testing.T) {
	page, err := os.ReadFile("testdata/page.txt")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("testdata/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Windows []struct {
			Text string `json:"text"`
		} `json:"windows"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	got := Windows(string(page))
	if len(got) != len(ref.Windows) {
		t.Fatalf("got %d windows, reference has %d", len(got), len(ref.Windows))
	}
	for i, w := range ref.Windows {
		if got[i] != w.Text {
			t.Fatalf("window %d differs:\n got  %q\n want %q", i, got[i], w.Text)
		}
	}
}

func TestWindowsOverlapAndLimit(t *testing.T) {
	ws := Windows(words(450))
	if len(ws) != 3 {
		t.Fatalf("450 words: got %d windows, want 3", len(ws))
	}
	for i, start := range []int{0, 150, 300} {
		if first := strings.Fields(ws[i])[0]; first != fmt.Sprintf("w%d", start) {
			t.Errorf("window %d starts at %s, want w%d", i, first, start)
		}
	}
	if got := len(Windows(words(10))); got != 1 {
		t.Errorf("10 words: got %d windows, want 1", got)
	}
	if got := len(Windows(words(50000))); got != MaxWindows {
		t.Errorf("50000 words: got %d windows, want %d", got, MaxWindows)
	}
	if got := Windows(""); len(got) != 0 {
		t.Errorf("empty text: got %d windows, want 0", len(got))
	}
}

func TestMeanNormalises(t *testing.T) {
	m := Mean([][]float64{{1, 0}, {0, 1}})
	want := math.Sqrt2 / 2
	if len(m) != 2 || math.Abs(m[0]-want) > 1e-12 || math.Abs(m[1]-want) > 1e-12 {
		t.Fatalf("got %v, want [%v %v]", m, want, want)
	}
}

// fakeEnc maps each text to a fixed unit vector and records its inputs.
type fakeEnc struct {
	vecs  map[string][]float64
	calls []string
}

func (f *fakeEnc) Encode(_ context.Context, text string) ([]float64, error) {
	f.calls = append(f.calls, text)
	v, ok := f.vecs[text]
	if !ok {
		return nil, fmt.Errorf("unexpected input %q", text)
	}
	return v, nil
}

func TestScoreUsesTheRecipe(t *testing.T) {
	page := words(10)
	enc := &fakeEnc{vecs: map[string][]float64{
		QueryInstruction + "kw": {1, 0},
		page:                    {0, 1},
		"alpha":                 {1, 0},
		"beta":                  {math.Sqrt2 / 2, math.Sqrt2 / 2},
	}}
	s := &Scorer{Enc: enc}
	kw, pg, err := s.Score(context.Background(), "kw", page, []string{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	wantCalls := []string{QueryInstruction + "kw", page, "alpha", "beta"}
	if strings.Join(enc.calls, "|") != strings.Join(wantCalls, "|") {
		t.Fatalf("encoder inputs %q, want %q", enc.calls, wantCalls)
	}
	wantKW := []float64{1, math.Sqrt2 / 2}
	wantPG := []float64{0, math.Sqrt2 / 2}
	for i := range wantKW {
		if math.Abs(kw[i]-wantKW[i]) > 1e-12 || math.Abs(pg[i]-wantPG[i]) > 1e-12 {
			t.Errorf("label %d: kw %v pg %v, want %v %v", i, kw[i], pg[i], wantKW[i], wantPG[i])
		}
	}
	if s.Model() != Model {
		t.Errorf("Model() = %q", s.Model())
	}
}

func TestScoreRejectsAnEmptyPage(t *testing.T) {
	s := &Scorer{Enc: &fakeEnc{}}
	if _, _, err := s.Score(context.Background(), "kw", "  \n ", []string{"a"}); err == nil {
		t.Fatal("want an error for a page with no words")
	}
}

func TestSimilarityUsesTheQueryInstruction(t *testing.T) {
	enc := &fakeEnc{vecs: map[string][]float64{QueryInstruction + "price action": {1, 0}, "Doji: Common pattern": {0.6, 0.8}, "Gap": {0, 1}}}
	s := &Scorer{Enc: enc}
	got, err := s.Similarity(context.Background(), "price action", []string{"Doji: Common pattern", "Gap"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 0.6 || got[1] != 0 {
		t.Fatalf("similarity = %v, want [0.6 0]", got)
	}
	wantCalls := []string{QueryInstruction + "price action", "Doji: Common pattern", "Gap"}
	if strings.Join(enc.calls, "|") != strings.Join(wantCalls, "|") {
		t.Fatalf("encoder inputs %q, want %q", enc.calls, wantCalls)
	}
}

func TestSimilarityOfNoTexts(t *testing.T) {
	enc := &fakeEnc{}
	got, err := (&Scorer{Enc: enc}).Similarity(context.Background(), "x", nil)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v; want nil, nil", got, err)
	}
	if len(enc.calls) != 0 {
		t.Fatalf("encoded %q, want nothing", enc.calls)
	}
}
