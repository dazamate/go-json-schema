package vocabtool

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// The golden cases are every Check and Vocabulary call the package's tests
// made before the checks were split into one check per file, with the report
// each gave. Replaying them holds the split to the same findings: the issues
// as a set (their order follows the checks now), and the counts, page
// elements and script tag exactly. A field holding "@file:<path>" holds that
// testdata file's text.

type goldenCase struct {
	Draft  string       `json:"draft"`
	Opts   CheckOptions `json:"opts"`
	Report Report       `json:"report"`
	Err    string       `json:"err,omitempty"`
}

// vocabularyOnly marks a case recorded from Vocabulary, not Check.
const vocabularyOnly = -1

func goldenCases(t *testing.T) []goldenCase {
	t.Helper()
	f, err := os.Open("testdata/golden/cases.jsonl.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	file := func(s string) string {
		p, ok := strings.CutPrefix(s, "@file:")
		if !ok {
			return s
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	var out []goldenCase
	sc := bufio.NewScanner(zr)
	sc.Buffer(nil, 64<<20)
	for sc.Scan() {
		var c goldenCase
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		c.Draft = file(c.Draft)
		c.Opts.PageHTML = file(c.Opts.PageHTML)
		c.Opts.PageText = file(c.Opts.PageText)
		out = append(out, c)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func sortedIssues(issues []Issue) []Issue {
	out := slices.Clone(issues)
	if out == nil {
		out = []Issue{}
	}
	slices.SortFunc(out, func(a, b Issue) int {
		return strings.Compare(a.Path+"\x00"+a.Severity+"\x00"+a.Message, b.Path+"\x00"+b.Severity+"\x00"+b.Message)
	})
	return out
}

func TestGolden(t *testing.T) {
	tool := newTool(t)
	cases := goldenCases(t)
	if len(cases) == 0 {
		t.Fatal("no golden cases")
	}
	for i, c := range cases {
		var got Report
		var err error
		if c.Opts.KeywordDensity == vocabularyOnly {
			got.Issues, err = tool.Vocabulary(json.RawMessage(c.Draft))
		} else {
			got, err = tool.Check(json.RawMessage(c.Draft), c.Opts)
		}
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		if msg != c.Err {
			t.Errorf("case %d: error %q, want %q", i, msg, c.Err)
			continue
		}
		want, have := sortedIssues(c.Report.Issues), sortedIssues(got.Issues)
		if !reflect.DeepEqual(have, want) {
			for _, w := range want {
				if !slices.Contains(have, w) {
					t.Errorf("case %d: lost %s %s: %s", i, w.Path, w.Severity, w.Message)
				}
			}
			for _, h := range have {
				if !slices.Contains(want, h) {
					t.Errorf("case %d: new %s %s: %s", i, h.Path, h.Severity, h.Message)
				}
			}
			if len(have) != len(want) {
				t.Errorf("case %d: %d issues, want %d", i, len(have), len(want))
			}
		}
		if got.Errors != c.Report.Errors || got.Warnings != c.Report.Warnings {
			t.Errorf("case %d: %d errors and %d warnings, want %d and %d", i, got.Errors, got.Warnings, c.Report.Errors, c.Report.Warnings)
		}
		if (len(got.PageElements) > 0 || len(c.Report.PageElements) > 0) && !reflect.DeepEqual(got.PageElements, c.Report.PageElements) {
			t.Errorf("case %d: page elements\n got %v\nwant %v", i, got.PageElements, c.Report.PageElements)
		}
		if got.ScriptTag != c.Report.ScriptTag {
			t.Errorf("case %d: script tag differs", i)
		}
	}
}
