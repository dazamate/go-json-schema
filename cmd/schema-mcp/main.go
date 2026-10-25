// Command schema-mcp is an MCP server for building entity-stuffed
// schema.org JSON-LD: SERP, page-term and entity research through the
// ChromiumFish browser, plus offline schema.org vocabulary tools.
//
// Claude Code (stdio):
//
//	claude mcp add schema -- schema-mcp
//
// Streamable HTTP, e.g. for n8n:
//
//	schema-mcp -http 127.0.0.1:8765
//
// API keys, such as TextRazor's for entity linking, live in a YAML file:
// ~/.config/go-json-schema/schema-mcp.yml by default (-config or
// SCHEMA_MCP_CONFIG to change it). See schema-mcp.yml.dist.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dazamate/go-json-schema/internal/azurelang"
	"github.com/dazamate/go-json-schema/internal/browser"
	"github.com/dazamate/go-json-schema/internal/cooldown"
	"github.com/dazamate/go-json-schema/internal/embed"
	"github.com/dazamate/go-json-schema/internal/mcpserver"
	"github.com/dazamate/go-json-schema/internal/research"
	"github.com/dazamate/go-json-schema/internal/textrazor"
	"github.com/dazamate/go-json-schema/internal/vocabtool"
)

const version = "0.1.0"

func main() {
	httpAddr := flag.String("http", "", "serve streamable HTTP on this address, e.g. 127.0.0.1:8765, instead of stdio")
	seed := flag.String("seed", envOr("SCHEMA_MCP_PERSONA_SEED", "go-json-schema"), "ChromiumFish fingerprint persona seed")
	cacheDir := flag.String("cache", envOr("SCHEMA_MCP_CACHE_DIR", defaultCacheDir()), "cache directory; empty disables caching")
	googleWait := flag.Duration("google-cooldown", envDuration("SCHEMA_MCP_GOOGLE_COOLDOWN", 30*time.Second), "least time between requests to Google, shared by every schema-mcp process; 0 disables")
	configPath := flag.String("config", envOr("SCHEMA_MCP_CONFIG", defaultConfigPath()), "settings file holding API keys (YAML)")
	flag.Parse()
	log.SetPrefix("schema-mcp: ")

	explicit := *configPath != defaultConfigPath()
	cfg, warnings, err := loadConfig(*configPath, explicit)
	if err != nil {
		log.Fatal(err)
	}
	for _, w := range warnings {
		log.Print(w)
	}
	if err := run(*httpAddr, *seed, *cacheDir, *googleWait, cfg); err != nil {
		log.Fatal(err)
	}
}

func run(httpAddr, seed, cacheDir string, googleWait time.Duration, cfg Config) error {
	vocab, err := vocabtool.New()
	if err != nil {
		return err
	}
	b := browser.New(seed)
	defer b.Close()
	// The cooldown's state stays in the default cache dir whatever -cache
	// says, so every schema-mcp process of this user shares it.
	state := ""
	if dir := defaultCacheDir(); dir != "" {
		state = filepath.Join(dir, "google-cooldown")
	}
	res := &research.Researcher{Browser: b, Google: cooldown.New(googleWait, state)}
	if cfg.TextRazor.APIKey != "" {
		res.Linker = &textrazor.Client{Key: cfg.TextRazor.APIKey}
		res.LinkMaxBytes = cfg.TextRazor.MaxKBPerPage * 1024
	}
	if cfg.AzureLanguage.Endpoint != "" {
		res.Bing = &azurelang.Client{Endpoint: cfg.AzureLanguage.Endpoint, Key: cfg.AzureLanguage.APIKey}
	}
	// The local model is fetched into the user cache dir on first use.
	if dir := defaultCacheDir(); dir != "" {
		enc := embed.NewONNX(filepath.Join(dir, "models"), research.Progress)
		defer enc.Close()
		res.Themes = &embed.Scorer{Enc: enc}
	}
	if cacheDir != "" {
		res.Cache = research.NewCache(cacheDir)
	}
	server := mcpserver.New(vocab, res, version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if httpAddr == "" {
		return server.Run(ctx, &mcp.StdioTransport{})
	}
	srv := &http.Server{Addr: httpAddr, Handler: mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	log.Printf("listening on %s", httpAddr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Fatalf("schema-mcp: %s: %v", key, err)
	}
	return d
}

func defaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "go-json-schema")
}
