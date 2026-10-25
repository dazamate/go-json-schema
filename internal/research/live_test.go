package research

import (
	"context"
	"os"
	"path/filepath"
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
