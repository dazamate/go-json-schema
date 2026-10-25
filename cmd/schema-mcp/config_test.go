package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "schema-mcp.yml")
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigReadsTheTextRazorSettings(t *testing.T) {
	path := writeConfig(t, "textrazor:\n  api_key: abc123\n  max_kb_per_page: 20\n", 0o600)
	cfg, warnings, err := loadConfig(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TextRazor.APIKey != "abc123" || cfg.TextRazor.MaxKBPerPage != 20 || len(warnings) != 0 {
		t.Fatalf("config = %+v, warnings %q", cfg, warnings)
	}
}

func TestLoadConfigRejectsUnknownKeys(t *testing.T) {
	path := writeConfig(t, "textrazor:\n  apikey: abc123\n", 0o600)
	_, _, err := loadConfig(path, true)
	if err == nil || !strings.Contains(err.Error(), "apikey") {
		t.Fatalf("err = %v, want the misspelt key named", err)
	}
	if strings.Contains(err.Error(), "abc123") {
		t.Fatalf("error shows the key's value: %v", err)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "none.yml")
	if cfg, _, err := loadConfig(path, false); err != nil || cfg.TextRazor.APIKey != "" {
		t.Fatalf("optional missing file: %+v, %v", cfg, err)
	}
	if _, _, err := loadConfig(path, true); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("required missing file: err = %v", err)
	}
}

func TestLoadConfigAcceptsAnEmptyFile(t *testing.T) {
	if _, _, err := loadConfig(writeConfig(t, "# nothing yet\n", 0o600), true); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigWarnsWhenOthersCanRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no Unix permission bits")
	}
	_, warnings, err := loadConfig(writeConfig(t, "textrazor:\n  api_key: k\n", 0o644), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "chmod 600") {
		t.Fatalf("warnings = %q", warnings)
	}
}

func TestExampleConfigLoads(t *testing.T) {
	cfg, _, err := loadConfig("schema-mcp.yml.dist", true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TextRazor.APIKey != "" {
		t.Fatal("the example config holds a key")
	}
}
