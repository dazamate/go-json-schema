package vocabtool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

// TestCheckConcurrent runs Check on one page from several goroutines at once
// with one Tool: each report must be the one a serial run gives.
func TestCheckConcurrent(t *testing.T) {
	tool := newTool(t)
	draft, err := os.ReadFile(filepath.Join("testdata", "lmb", "2-movers-cost.jsonld"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join("testdata", "lmb", "2-movers-cost.html"))
	if err != nil {
		t.Fatal(err)
	}
	opts := CheckOptions{PageHTML: string(page), PageURL: "https://liftmyboxes.com/2-movers-cost/", Keyword: "2 movers cost"}
	const runs = 8
	got := make([]Report, runs)
	errs := make([]error, runs)
	var wg sync.WaitGroup
	for i := range runs {
		wg.Go(func() { got[i], errs[i] = tool.Check(draft, opts) })
	}
	wg.Wait()
	want, err := tool.Check(draft, opts)
	if err != nil {
		t.Fatal(err)
	}
	for i := range runs {
		if errs[i] != nil {
			t.Errorf("run %d: %v", i, errs[i])
		} else if !reflect.DeepEqual(got[i], want) {
			t.Errorf("run %d: report differs from a serial run", i)
		}
	}
}

// TestVocabularyArrayRootPaths: Vocabulary checks an array root as given, so
// its nodes keep their $[i] paths.
func TestVocabularyArrayRootPaths(t *testing.T) {
	got, err := newTool(t).Vocabulary(json.RawMessage(`[{"@type":"Thng"}]`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Issue{{Path: "$[0]", Severity: Error, Message: `unknown type "Thng"; did you mean Thing?`}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNormalizeMatchesWalk: the context's normalized draft, before any check
// runs, is the draft today's walk published, for every recorded Check call.
func TestNormalizeMatchesWalk(t *testing.T) {
	tool := newTool(t)
	for i, c := range goldenCases(t) {
		if c.Opts.KeywordDensity == vocabularyOnly || c.Err != "" {
			continue
		}
		ctx, err := audit.NewContext(tool.Vocab, []byte(c.Draft), input(c.Opts))
		if err != nil {
			t.Errorf("case %d: %v", i, err)
			continue
		}
		tag, err := scriptTag(ctx.Root, c.Opts.Pretty)
		if err != nil {
			t.Errorf("case %d: %v", i, err)
			continue
		}
		if tag != c.Report.ScriptTag {
			t.Errorf("case %d: normalized draft differs from the walk's\n got %s\nwant %s", i, tag, c.Report.ScriptTag)
		}
	}
}
