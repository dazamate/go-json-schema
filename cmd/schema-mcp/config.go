package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"go.yaml.in/yaml/v3"
)

// Config is the schema-mcp settings file: API keys for the outside APIs it
// can call. See schema-mcp.yml.dist.
type Config struct {
	TextRazor     TextRazorConfig     `yaml:"textrazor"`
	AzureLanguage AzureLanguageConfig `yaml:"azure_language"`
}

// AzureLanguageConfig enables Bing entity IDs in select_entities, through
// Azure AI Language entity linking. Endpoint and APIKey are both set or both
// empty.
type AzureLanguageConfig struct {
	Endpoint string `yaml:"endpoint"` // https URL of the Language resource
	APIKey   string `yaml:"api_key"`
}

// TextRazorConfig enables entity linking in page_terms.
type TextRazorConfig struct {
	APIKey       string `yaml:"api_key"`
	MaxKBPerPage int    `yaml:"max_kb_per_page"` // text sent per page; default 30
}

// defaultConfigPath is schema-mcp.yml in the user's config dir, e.g.
// ~/.config/go-json-schema/schema-mcp.yml on Linux.
func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "go-json-schema", "schema-mcp.yml")
}

// loadConfig reads the settings file. A missing file is an error only when
// required (the path was given explicitly). Unknown keys are errors, so a
// misspelt key name fails at startup instead of silently disabling an API.
// warnings says when users other than the owner can read the file.
func loadConfig(path string, required bool) (Config, []string, error) {
	var cfg Config
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && !required {
			return cfg, nil, nil
		}
		return cfg, nil, err
	}
	defer f.Close()
	var warnings []string
	if info, err := f.Stat(); err == nil && runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		warnings = append(warnings, fmt.Sprintf("%s holds API keys and other users can read it; run chmod 600 %[1]s", path))
	}
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.TextRazor.MaxKBPerPage < 0 {
		return Config{}, nil, fmt.Errorf("%s: textrazor.max_kb_per_page is negative", path)
	}
	// The key is never put in these errors.
	az := cfg.AzureLanguage
	if (az.Endpoint == "") != (az.APIKey == "") {
		return Config{}, nil, fmt.Errorf("%s: azure_language needs both endpoint and api_key, or neither", path)
	}
	if az.Endpoint != "" {
		if u, err := url.Parse(az.Endpoint); err != nil || u.Scheme != "https" || u.Host == "" {
			return Config{}, nil, fmt.Errorf("%s: azure_language.endpoint must be an https URL", path)
		}
	}
	return cfg, warnings, nil
}
