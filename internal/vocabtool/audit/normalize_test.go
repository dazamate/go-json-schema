package audit

import (
	"sync"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

var (
	testVocabOnce sync.Once
	testVocabV    *vocab.Vocab
	testVocabErr  error
)

// testVocab loads the embedded vocabulary once for the package's tests.
func testVocab(t *testing.T) *vocab.Vocab {
	t.Helper()
	testVocabOnce.Do(func() { testVocabV, testVocabErr = vocab.Load() })
	if testVocabErr != nil {
		t.Fatal(testVocabErr)
	}
	return testVocabV
}

// normalized decodes the draft, normalizes it and encodes it again.
func normalized(t *testing.T, draft string) string {
	t.Helper()
	root, err := jsonld.Decode([]byte(draft))
	if err != nil {
		t.Fatal(err)
	}
	Normalize(testVocab(t), root)
	b, err := jsonld.Encode(root, false)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestNormalizeExpandsEnum(t *testing.T) {
	got := normalized(t, `{"@type":"Offer","availability":"InStock"}`)
	if want := `{"@type":"Offer","availability":"https://schema.org/InStock"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestNormalizeLeavesUnknown(t *testing.T) {
	draft := `{"@type":"Offer","availability":"Nope"}`
	if got := normalized(t, draft); got != draft {
		t.Errorf("got %s, want %s", got, draft)
	}
}

func TestNormalizeSkipsForeignKeys(t *testing.T) {
	draft := `{"@type":"Offer","ex:availability":"InStock"}`
	if got := normalized(t, draft); got != draft {
		t.Errorf("got %s, want %s", got, draft)
	}
}

func TestNormalizeUntypedNested(t *testing.T) {
	got := normalized(t, `{"@graph":[{"@type":"Product","offers":[{"availability":"InStock"}]}]}`)
	if want := `{"@graph":[{"@type":"Product","offers":[{"availability":"https://schema.org/InStock"}]}]}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
