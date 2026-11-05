// Package audittest runs one audit check over a draft, for the per-check
// tests. It builds the context the way Check does, so the draft is
// normalized and the page read as in production, and it imports every check
// area, so the ledgers see every check.
package audittest

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/all" // register every check
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

// Input is a test case's draft and what goes with it. Only Draft is needed.
type Input struct {
	Draft          string
	PageHTML       string
	PageText       string
	PageURL        string
	Keyword        string
	KeywordDensity float64
}

var (
	loadOnce sync.Once
	loaded   *vocab.Vocab
	loadErr  error
)

// vocabulary is the embedded vocabulary, loaded once for every test.
func vocabulary(t testing.TB) *vocab.Vocab {
	t.Helper()
	loadOnce.Do(func() { loaded, loadErr = vocab.Load() })
	if loadErr != nil {
		t.Fatalf("audittest: load vocabulary: %v", loadErr)
	}
	return loaded
}

// Run builds the context for the input and runs the check alone over it,
// returning its issues in the order it found them.
func Run(t testing.TB, c audit.Check, in Input) []audit.Issue {
	t.Helper()
	ctx, err := audit.NewContext(vocabulary(t), []byte(in.Draft), audit.Input{
		PageText:       in.PageText,
		PageHTML:       in.PageHTML,
		PageURL:        in.PageURL,
		Keyword:        in.Keyword,
		KeywordDensity: in.KeywordDensity,
	})
	if err != nil {
		t.Fatalf("audittest: %s: %v", c.ID(), err)
	}
	return c.Run(ctx)
}

// Expect runs the check as Run does and fails the test unless it found
// exactly want, in order. No issues and an empty want are equal.
func Expect(t testing.TB, c audit.Check, in Input, want []audit.Issue) {
	t.Helper()
	got := Run(t, c, in)
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %s\nwant %s", c.ID(), list(got), list(want))
	}
}

// list writes issues one to a line, for a failure message.
func list(issues []audit.Issue) string {
	if len(issues) == 0 {
		return "no issues"
	}
	var b strings.Builder
	for _, i := range issues {
		fmt.Fprintf(&b, "\n  %s %s: %s", i.Path, i.Severity, i.Message)
	}
	return b.String()
}
