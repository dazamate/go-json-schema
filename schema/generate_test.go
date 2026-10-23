package schema

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/schemagen"
)

// TestGeneratedFilesUpToDate regenerates the package next to a copy of the
// handwritten files. It fails when the *_gen.go files are stale or when a
// handwritten name clashes with a generated one. Fix with go generate.
func TestGeneratedFilesUpToDate(t *testing.T) {
	dir := t.TempDir()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_gen.go") || strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	vocab, err := schemagen.Load(vocabPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := schemagen.Generate(vocab, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	gen, _ := filepath.Glob(filepath.Join(dir, "*_gen.go"))
	for _, f := range gen {
		fresh, _ := os.ReadFile(f)
		current, err := os.ReadFile(filepath.Base(f))
		if err != nil {
			t.Fatalf("%s missing: run go generate ./schema", filepath.Base(f))
		}
		if !bytes.Equal(fresh, current) {
			t.Fatalf("%s is stale: run go generate ./schema", filepath.Base(f))
		}
	}
}
