package textrazor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sample = `{"ok":true,"time":0.1,"response":{"language":"eng",
 "entities":[{"entityId":"Moving company","entityEnglishId":"Moving company","wikiLink":"http://en.wikipedia.org/wiki/Moving_company",
   "wikidataId":"Q1129587","freebaseId":"/m/0bwxkd","matchedText":"movers","relevanceScore":0.71,"confidenceScore":4.2,"type":["Company"]}],
 "topics":[{"id":0,"label":"Relocation (personal)","wikiLink":"http://en.wikipedia.org/Category:Relocation","wikidataId":"Q2918584","score":0.9}]}}`

func TestAnalyzePostsTextAndDecodesEntitiesAndTopics(t *testing.T) {
	var got http.Header
	var form map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		r.ParseForm()
		form = r.PostForm
		w.Write([]byte(sample))
	}))
	defer srv.Close()
	c := &Client{Key: "secret", Endpoint: srv.URL}
	res, err := c.Analyze(context.Background(), "Hire movers to move house.")
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("X-TextRazor-Key") != "secret" {
		t.Errorf("key header = %q", got.Get("X-TextRazor-Key"))
	}
	if form["text"][0] != "Hire movers to move house." || form["extractors"][0] != "entities,topics" || form["cleanup.mode"][0] != "raw" {
		t.Errorf("form = %v", form)
	}
	if _, ok := form["url"]; ok {
		t.Errorf("sent a url; TextRazor must never fetch pages itself")
	}
	if len(res.Entities) != 1 || res.Entities[0].WikidataID != "Q1129587" || res.Entities[0].FreebaseID != "/m/0bwxkd" ||
		res.Entities[0].MatchedText != "movers" || res.Entities[0].Relevance != 0.71 || res.Entities[0].Confidence != 4.2 {
		t.Errorf("entities = %+v", res.Entities)
	}
	if len(res.Topics) != 1 || res.Topics[0].Label != "Relocation (personal)" || res.Topics[0].Score != 0.9 {
		t.Errorf("topics = %+v", res.Topics)
	}
	if res.Units != 1 {
		t.Errorf("units = %d, want 1 for under 10 KB", res.Units)
	}
}

func TestAnalyzeErrorNeverEchoesTheKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Your TextRazor API key is invalid or has used up its quota.", http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := &Client{Key: "secret-key-123", Endpoint: srv.URL}
	_, err := c.Analyze(context.Background(), "text")
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "quota") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "secret-key-123") {
		t.Fatalf("error leaks the key: %v", err)
	}
}

func TestAnalyzeReportsAnUnsuccessfulResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"error":"Text too short"}`))
	}))
	defer srv.Close()
	_, err := (&Client{Key: "k", Endpoint: srv.URL}).Analyze(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "Text too short") {
		t.Fatalf("err = %v", err)
	}
}

func TestClipKeepsWholeRunesAndWords(t *testing.T) {
	text := strings.Repeat("ab ", 10) + "café"
	got := Clip(text, 9)
	if got != "ab ab ab" {
		t.Fatalf("Clip = %q", got)
	}
	if Clip("short", 100) != "short" {
		t.Fatal("short text changed")
	}
	if got := Clip("aé", 2); got != "a" {
		t.Fatalf("Clip split a rune: %q", got)
	}
}

func TestUnitsCountsTenKBBlocks(t *testing.T) {
	for n, want := range map[int]int{0: 1, 1: 1, 10240: 1, 10241: 2, 30000: 3} {
		if got := Units(n); got != want {
			t.Errorf("Units(%d) = %d, want %d", n, got, want)
		}
	}
}
