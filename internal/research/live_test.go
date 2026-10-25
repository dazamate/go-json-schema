package research

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dazamate/go-json-schema/internal/browser"
	"github.com/dazamate/go-json-schema/internal/cooldown"
)

// TestLive runs the research tools against the real web through
// ChromiumFish. It needs chromiumfish installed:
//
//	SCHEMA_MCP_LIVE=1 go test ./internal/research -run TestLive -v
func TestLive(t *testing.T) {
	if os.Getenv("SCHEMA_MCP_LIVE") == "" {
		t.Skip("set SCHEMA_MCP_LIVE=1 to run against the real web")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	b := browser.New("go-json-schema-test")
	defer b.Close()
	// The same shared cooldown schema-mcp uses, so the test never bursts Google.
	state := ""
	if dir, err := os.UserCacheDir(); err == nil {
		state = filepath.Join(dir, "go-json-schema", "google-cooldown")
	}
	r := &Researcher{Browser: b, Google: cooldown.New(30*time.Second, state)}

	s, err := r.SERP(ctx, SERPQuery{Keyword: "lemon herb chicken"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("serp: engine=%s results=%d paa=%d related=%d autocomplete=%d note=%q", s.Engine, len(s.Results), len(s.PeopleAlsoAsk), len(s.RelatedSearches), len(s.Autocomplete), s.Note)
	if len(s.Results) < 3 {
		t.Fatalf("only %d results", len(s.Results))
	}
	for _, res := range s.Results {
		t.Logf("  %d %s | %s | %s", res.Rank, res.Domain, res.Title, res.URL)
	}

	var links []string
	for _, res := range s.Results[:3] {
		links = append(links, res.Link)
	}
	rep, err := r.PageTerms(ctx, PageQuery{Links: links, MaxTerms: 15})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rep.Pages {
		t.Logf("page: %s words=%d headings=%d schema=%v err=%q", p.URL, p.WordCount, len(p.Headings), p.Schema != nil, p.Error)
	}
	t.Logf("words: %v", rep.Words)
	t.Logf("phrases: %v", rep.Phrases)
	t.Logf("schema terms: %v", rep.SchemaTerms)
	if len(rep.Phrases) == 0 {
		t.Fatal("no phrases")
	}

	ents, err := r.ResolveEntities(ctx, EntityQuery{Phrases: []string{"lemon", "rosemary", "air fryer", "chicken thigh", "mercury", "lemon herb chicken", "dijon mustard"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents.Entities {
		t.Logf("entity: %-18s %-9s %-22s %-10s kgmid=%s from=%s types=%v candidates=%d", e.Phrase, e.Status, e.Title, e.QID, e.KGMID, e.KGMIDFrom, e.Types, len(e.Candidates))
	}
	if ents.Entities[0].KGMID == "" {
		t.Fatal("lemon has no kgmid")
	}
}

// TestLivePageJSReadsMainLinks serves the calibration page locally and reads
// it with the real browser: pageJS keeps the links in <main> and drops the
// ones in the hidden header and footer.
//
//	SCHEMA_MCP_LIVE=1 go test ./internal/research -run LivePageJS -v
func TestLivePageJSReadsMainLinks(t *testing.T) {
	if os.Getenv("SCHEMA_MCP_LIVE") == "" {
		t.Skip("set SCHEMA_MCP_LIVE=1 to run against the real browser")
	}
	html, err := os.ReadFile(filepath.Join("testdata", "select", "page.html"))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/moving-cost-calculator/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(html)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	b := browser.New("go-json-schema-test")
	defer b.Close()
	r := &Researcher{Browser: b}
	var c cachedPage
	err = b.Use(ctx, func(p browser.Page) error {
		c = r.fetchPage(ctx, p, srv.URL+"/moving-cost-calculator/", "")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Page.Error != "" {
		t.Fatalf("fetchPage: %s", c.Page.Error)
	}
	has := func(suffix string) *pageLink {
		for i, l := range c.Links {
			u, err := url.Parse(l.Href)
			if err == nil && u.Host == host && strings.HasSuffix(u.Path, suffix) {
				return &c.Links[i]
			}
		}
		return nil
	}
	for _, s := range []string{"/cost-to-move-a-pool-table/", "/cost-to-move-a-piano/", "/cost-to-move-a-hot-tub/"} {
		if has(s) == nil {
			t.Errorf("missing main link %s in %+v", s, c.Links)
		}
	}
	if l := has("/cost-to-move-a-pool-table/"); l != nil && !strings.Contains(l.Text, "Pool table") {
		t.Errorf("pool table link text = %q", l.Text)
	}
	for _, s := range []string{"/privacy-policy/", "/about-us/", "/terms-and-conditions/"} {
		if has(s) != nil {
			t.Errorf("chrome link %s kept", s)
		}
	}
	t.Logf("links: %+v", c.Links)
}
