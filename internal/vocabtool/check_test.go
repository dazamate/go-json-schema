package vocabtool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
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

// TestNormalizeMatchesWalkOn holds Normalize to the walk's rewrite on the
// drafts the golden cases do not reach: array roots, untyped nodes, @graph,
// keys that are not schema.org terms, Google's annotations, unknown and
// mixed-range properties, nested arrays and other spellings of a member's
// URL. Walked is the draft as the vocabulary walk published it, recorded
// before the walk moved to the terms checks, which no longer rewrite.
// Rewrites says whether the walk changes the draft at all.
func TestNormalizeMatchesWalkOn(t *testing.T) {
	tool := newTool(t)
	for _, tc := range []struct {
		name, draft, walked string
		rewrites            bool
	}{
		{"array root", `[{"@type":"Offer","availability":"InStock"},{"@type":"Offer","availability":"OutOfStock"}]`,
			`[{"@type":"Offer","availability":"https://schema.org/InStock"},{"@type":"Offer","availability":"https://schema.org/OutOfStock"}]`, true},
		{"untyped node under @graph", `{"@graph":[{"availability":"InStock"}]}`,
			`{"@graph":[{"availability":"https://schema.org/InStock"}]}`, true},
		{"@graph in a property value", `{"@type":"WebPage","mainEntity":{"@graph":[{"availability":"InStock"}]}}`,
			`{"@type":"WebPage","mainEntity":{"@graph":[{"availability":"https://schema.org/InStock"}]}}`, true},
		{"untyped node in a property value", `{"@type":"Product","offers":{"availability":"InStock"}}`,
			`{"@type":"Product","offers":{"availability":"https://schema.org/InStock"}}`, true},
		{"prefixed and foreign keys", `{"@type":"Offer","schema:availability":"InStock","https://schema.org/availability":"InStock","ex:availability":"InStock"}`,
			`{"@type":"Offer","schema:availability":"https://schema.org/InStock","https://schema.org/availability":"https://schema.org/InStock","ex:availability":"InStock"}`, true},
		{"foreign key alone", `{"@type":"Offer","ex:availability":"InStock","ex:offers":{"@type":"Offer","availability":"InStock"}}`,
			`{"@type":"Offer","ex:availability":"InStock","ex:offers":{"@type":"Offer","availability":"InStock"}}`, false},
		{"Google annotation", `{"@type":"SearchAction","query-input":{"@type":"Offer","availability":"InStock"},"target-output":"InStock"}`,
			`{"@type":"SearchAction","query-input":{"@type":"Offer","availability":"InStock"},"target-output":"InStock"}`, false},
		{"unknown property", `{"@type":"Product","fooBar":{"@type":"Offer","availability":"InStock"}}`,
			`{"@type":"Product","fooBar":{"@type":"Offer","availability":"InStock"}}`, false},
		{"mixed range", `{"@type":"ItemList","itemListOrder":"Ascending"}`,
			`{"@type":"ItemList","itemListOrder":"Ascending"}`, false},
		{"mixed range as a URL", `{"@type":"ItemList","itemListOrder":"https://schema.org/ItemListOrderAscending"}`,
			`{"@type":"ItemList","itemListOrder":"https://schema.org/ItemListOrderAscending"}`, false},
		{"not a member", `{"@type":"Offer","availability":"Nope"}`,
			`{"@type":"Offer","availability":"Nope"}`, false},
		{"nested arrays", `{"@type":"Product","offers":[[{"@type":"Offer","availability":["InStock",["OutOfStock"]]}]]}`,
			`{"@type":"Product","offers":[[{"@type":"Offer","availability":["https://schema.org/InStock",["OutOfStock"]]}]]}`, true},
		{"other URL spellings", `{"@type":"Offer","availability":["http://schema.org/InStock","schema:OutOfStock","https://schema.org/PreOrder"]}`,
			`{"@type":"Offer","availability":["https://schema.org/InStock","https://schema.org/OutOfStock","https://schema.org/PreOrder"]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decode := func() any {
				root, err := jsonld.Decode([]byte(tc.draft))
				if err != nil {
					t.Fatal(err)
				}
				return root
			}
			encode := func(root any) string {
				b, err := jsonld.Encode(root, false)
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			orig, normal := decode(), decode()
			audit.Normalize(tool.Vocab, normal)
			if n := encode(normal); n != tc.walked {
				t.Errorf("Normalize differs from the walk\n walk %s\n norm %s", tc.walked, n)
			}
			if changed := tc.walked != encode(orig); changed != tc.rewrites {
				t.Errorf("walk rewrote the draft: %v, want %v: %s", changed, tc.rewrites, tc.walked)
			}
		})
	}
}
