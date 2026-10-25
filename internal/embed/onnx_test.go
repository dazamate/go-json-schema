//go:build cgo

package embed

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestONNXMatchesPythonReference runs the real model against what
// google-seo-docs' Python recipe recorded in testdata/reference.json. The
// first run downloads about 220 MB into the user cache.
func TestONNXMatchesPythonReference(t *testing.T) {
	if os.Getenv("SCHEMA_MCP_MODEL") != "1" {
		t.Skip("set SCHEMA_MCP_MODEL=1 to run the real model")
	}
	type text struct {
		Text string `json:"text"`
		IDs  []int  `json:"ids"`
	}
	var ref struct {
		Keyword string `json:"keyword"`
		Query   text   `json:"query"`
		Windows []text `json:"windows"`
		Probes  []text `json:"probes"`
		Themes  []struct {
			Label   string  `json:"label"`
			IDs     []int   `json:"ids"`
			Keyword float64 `json:"keyword"`
			Page    float64 `json:"page"`
		} `json:"themes"`
	}
	raw, err := os.ReadFile("testdata/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile("testdata/page.txt")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	o := NewONNX(filepath.Join(cache, "go-json-schema", "models"), func(_ context.Context, msg string) { t.Log(msg) })
	defer o.Close()
	ctx := context.Background()

	texts := append([]text{ref.Query}, ref.Windows...)
	texts = append(texts, ref.Probes...)
	for _, th := range ref.Themes {
		texts = append(texts, text{th.Label, th.IDs})
	}
	for _, tx := range texts {
		got, err := o.tokens(ctx, tx.Text)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, tx.IDs) {
			t.Fatalf("token ids differ for %.80q:\n got  %v\n want %v", tx.Text, got, tx.IDs)
		}
	}
	if n := len(ref.Probes[1].IDs); n != MaxTokens {
		t.Fatalf("the long probe has %d ids, want %d", n, MaxTokens)
	}

	labels := make([]string, len(ref.Themes))
	for i, th := range ref.Themes {
		labels[i] = th.Label
	}
	s := &Scorer{Enc: o}
	kw, pg, err := s.Score(ctx, ref.Keyword, string(page), labels)
	if err != nil {
		t.Fatal(err)
	}
	var maxKW, maxPG float64
	var maxKWLabel, maxPGLabel string
	for i, th := range ref.Themes {
		dk, dp := math.Abs(kw[i]-th.Keyword), math.Abs(pg[i]-th.Page)
		if dk > maxKW {
			maxKW, maxKWLabel = dk, th.Label
		}
		if dp > maxPG {
			maxPG, maxPGLabel = dp, th.Label
		}
		if dk > 2e-5 || dp > 2e-5 {
			t.Errorf("%s: keyword %.6f page %.6f, want %.6f %.6f", th.Label, kw[i], pg[i], th.Keyword, th.Page)
		}
		for _, cut := range []float64{0.40, 0.55} {
			if (kw[i] >= cut) != (th.Keyword >= cut) || (pg[i] >= cut) != (th.Page >= cut) {
				t.Errorf("%s: decision at %.2f differs: keyword %.6f page %.6f, want %.6f %.6f",
					th.Label, cut, kw[i], pg[i], th.Keyword, th.Page)
			}
		}
		if th.Label == "Freight transport" {
			t.Logf("Freight transport: keyword %.6f (reference %.6f)", kw[i], th.Keyword)
			if kw[i] < 0.40 {
				t.Errorf("Freight transport's keyword score %.6f does not pass 0.40", kw[i])
			}
		}
	}
	t.Logf("largest |Δ|: keyword %.2e (%s), page %.2e (%s)", maxKW, maxKWLabel, maxPG, maxPGLabel)
}
