package research

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dazamate/go-json-schema/internal/browser"
	"github.com/dazamate/go-json-schema/internal/embed"
	"github.com/dazamate/go-json-schema/internal/textrazor"
)

// The forex calibration page is the second-page check of the keyword-graph
// design: a page whose keyword has its own Wikipedia article.
const (
	forexURL     = "https://www.theforexguy.com/price-action-trading/"
	forexKeyword = "price action trading"
	forexDir     = "select-forex"
)

func forexFixturePath(name string) string { return filepath.Join("testdata", forexDir, name) }

func forexQuery() SelectQuery {
	return SelectQuery{URL: forexURL, Keyword: forexKeyword, Settings: DefaultSelectSettings}
}

// forexCosines is cosines.json: the keyword's cosine against each graph text.
type forexCosines struct {
	Keyword string             `json:"keyword"`
	Texts   map[string]float64 `json:"texts"`
}

// recordingLinker answers from the TextRazor cache, or from TextRazor when
// TEXTRAZOR_API_KEY is set, and keeps the result it returns.
type recordingLinker struct {
	cache *Cache

	mu    sync.Mutex
	texts map[string]bool
	res   textrazor.Result
}

func (l *recordingLinker) Analyze(ctx context.Context, text string) (textrazor.Result, error) {
	var res textrazor.Result
	if !l.cache.Get("textrazor", textrazorKey(text), pageTTL, &res) {
		key := os.Getenv("TEXTRAZOR_API_KEY")
		if key == "" {
			return textrazor.Result{}, errors.New("TextRazor answer not cached for this page text; set TEXTRAZOR_API_KEY")
		}
		var err error
		if res, err = (&textrazor.Client{Key: key}).Analyze(ctx, text); err != nil {
			return textrazor.Result{}, err
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.texts[text] = true
	l.res = res
	return res, nil
}

// recordingScorer scores with the real model and keeps every graph text's
// cosine.
type recordingScorer struct {
	embed.Scorer

	mu    sync.Mutex
	texts map[string]float64
}

func (s *recordingScorer) Similarity(ctx context.Context, keyword string, texts []string) ([]float64, error) {
	sims, err := s.Scorer.Similarity(ctx, keyword, texts)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, text := range texts {
		s.texts[text] = sims[i]
	}
	return sims, nil
}

// TestRecordForexFixtures records testdata/select-forex: it runs
// select_entities on the forex page with the real browser, Wikipedia and
// Wikidata and the real embedding model. The page text must be in the
// TextRazor cache, or TEXTRAZOR_API_KEY set. It needs chromiumfish and the
// model files installed:
//
//	SCHEMA_MCP_RECORD=1 go test ./internal/research -run RecordForex -v -timeout 15m
func TestRecordForexFixtures(t *testing.T) {
	if os.Getenv("SCHEMA_MCP_RECORD") == "" {
		t.Skip("set SCHEMA_MCP_RECORD=1 to re-record testdata/select-forex")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	cacheDir = filepath.Join(cacheDir, "go-json-schema")

	b := browser.New("go-json-schema-record")
	defer b.Close()
	rec := &wikiRecorder{t: t, b: b, fx: wikiFixture{
		Items: map[string]json.RawMessage{}, Links: map[string][]string{}, LinksHere: map[string][]string{}, Options: map[string][]string{},
		Pages: map[string]fixturePage{}, Redirects: map[string]string{},
	}}
	linker := &recordingLinker{cache: NewCache(cacheDir), texts: map[string]bool{}}
	scorer := &recordingScorer{Scorer: embed.Scorer{Enc: embed.NewONNX(filepath.Join(cacheDir, "models"), Progress)}, texts: map[string]float64{}}
	r := &Researcher{Browser: rec, Cache: NewCache(t.TempDir()), Linker: linker, Themes: scorer}
	rep, err := r.SelectEntities(ctx, forexQuery())
	if err != nil {
		t.Fatal(err)
	}
	if len(linker.texts) != 1 {
		t.Fatalf("TextRazor was asked about %d texts, want 1", len(linker.texts))
	}

	write := func(name string, v any) int {
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(forexFixturePath(""), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(forexFixturePath(name), append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return len(out) + 1
	}
	t.Logf("wrote page.json, %d bytes", write("page.json", rec.page))
	t.Logf("wrote textrazor.json, %d bytes", write("textrazor.json", linker.res))
	t.Logf("wrote cosines.json, %d texts, %d bytes", len(scorer.texts), write("cosines.json", forexCosines{Keyword: forexKeyword, Texts: scorer.texts}))
	t.Logf("wrote wiki.json, %d items, %d pages, %d redirects, %d pages with links, %d with in-links, %d bytes",
		len(rec.fx.Items), len(rec.fx.Pages), len(rec.fx.Redirects), len(rec.fx.Links), len(rec.fx.LinksHere), write("wiki.json", rec.fx))

	t.Logf("about: %v", selNames(rep.About))
	t.Logf("mentions: %v", selNames(rep.Mentions))
	for _, d := range rep.Dropped {
		links := -1
		if d.GraphLinks != nil {
			links = *d.GraphLinks
		}
		t.Logf("dropped: %s | %s | %s | graph_links %d | graph_similarity %v", d.Name, d.Reason, d.Source, links, num(d.GraphSimilarity))
	}
	g, _ := json.Marshal(rep.Graph)
	t.Logf("graph: %s", g)
	t.Logf("note: %s", rep.Note)
}

// forexResearcher replays the forex recording: the recorded TextRazor answer
// and cosines, with the recorded Wikipedia.
func forexResearcher(t *testing.T, page *fakePage) *Researcher {
	t.Helper()
	var res textrazor.Result
	if err := json.Unmarshal(readForexFixture(t, "textrazor.json"), &res); err != nil {
		t.Fatal(err)
	}
	var cos forexCosines
	if err := json.Unmarshal(readForexFixture(t, "cosines.json"), &cos); err != nil {
		t.Fatal(err)
	}
	if cos.Keyword != forexKeyword {
		t.Fatalf("cosines.json is for %q", cos.Keyword)
	}
	return &Researcher{
		Browser: fakeBrowser{page},
		Linker:  &fakeLinker{link: func(string) (textrazor.Result, error) { return res, nil }},
		Themes:  &fakeScorer{sim: cos.Texts},
	}
}

func readForexFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(forexFixturePath(name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestSelectEntitiesForexCalibration replays the forex page from the
// recorded fixtures.
func TestSelectEntitiesForexCalibration(t *testing.T) {
	if _, err := os.Stat(forexFixturePath("wiki.json")); errors.Is(err, fs.ErrNotExist) {
		t.Skip("record testdata/select-forex first: SCHEMA_MCP_RECORD=1 go test ./internal/research -run RecordForex")
	}
	w := loadWikiFixture(t, forexDir)
	var raw rawPage
	if err := json.Unmarshal(readForexFixture(t, "page.json"), &raw); err != nil {
		t.Fatal(err)
	}
	rep := runSelect(t, forexResearcher(t, selectPage(w, raw, nil)), forexQuery())
	defer func() {
		if b, err := json.MarshalIndent(rep, "", "  "); t.Failed() && err == nil {
			t.Logf("report: %s", b)
		}
	}()

	if strings.Contains(rep.Note, "graph not scored") {
		t.Errorf("note = %q", rep.Note)
	}
	wantAbout := []string{
		"Price action trading", "Technical analysis", "Market trend", "Candlestick pattern", "Scalping (trading)",
		"Stock market index", "Day trading", "Stock", "Trader (finance)", "Stock market", "Foreign exchange market", "Swing trading",
		"Broker", "Trading strategy", "Algorithmic trading", "Order (exchange)", "Price channels", "Financial quote", "Floor trader",
		"Volatility (finance)", "Stock market index future", "Gap (chart pattern)", "Chart pattern", "Forex signal",
		"Open-high-low-close chart", "Triangle (chart pattern)", "Volume (finance)", "Broadening top", "Commodity channel index",
		"Market liquidity", "Stock market data systems", "Wedge pattern", "Double top and double bottom", "Support and resistance",
		"Breakout (technical analysis)", "Head and shoulders (chart pattern)", "Candlestick chart", "Trend line (technical analysis)",
		"Doji", "Flag and pennant patterns", "Slippage (finance)", "Moving average",
	}
	wantMentions := []string{} // every page entity in the graph is already in about, or has fewer than 2 graph links
	wantGraph := GraphReport{Centres: []string{"Price action trading"}, OutLinks: 40, InLinks: 6, Neighbours: 44, Kept: 35, Cut: 9}
	if got := selNames(rep.About); !slices.Equal(got, wantAbout) {
		t.Errorf("about = %q\nwant    %q", got, wantAbout)
	}
	if got := selNames(rep.Mentions); !slices.Equal(got, wantMentions) {
		t.Errorf("mentions = %q\nwant       %q", got, wantMentions)
	}
	if rep.Graph == nil || !reflect.DeepEqual(*rep.Graph, wantGraph) {
		t.Errorf("graph = %+v\nwant   %+v", rep.Graph, wantGraph)
	}

	// These hold whatever the recording says.
	about := selNames(rep.About)
	for _, name := range []string{"Foreign exchange market", "Technical analysis", "Market trend", "Candlestick pattern"} {
		if !slices.Contains(about, name) {
			t.Errorf("%s is not in about", name)
		}
	}
	for _, name := range []string{"Candle", "Candlestick", "Signal"} {
		if d, ok := findDropped(rep, name); !ok || d.Reason != ReasonOffGraph {
			t.Errorf("%s dropped = %+v", name, d)
		}
	}
	for _, name := range []string{"ISBN", "One Direction"} {
		if slices.Contains(about, name) || slices.Contains(selNames(rep.Mentions), name) {
			t.Errorf("%s was selected", name)
		}
	}
}
