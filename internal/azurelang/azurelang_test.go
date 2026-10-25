package azurelang

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

// sample is Microsoft's documented entity-linking response (quickstart, REST).
const sample = `{"kind":"EntityLinkingResults","results":{"documents":[{"id":"1","entities":[{"bingId":"a093e9b9-90f5-a3d5-c4b8-5855e1b01f85","name":"Microsoft","matches":[{"text":"Microsoft","offset":0,"length":9,"confidenceScore":0.48}],"language":"en","id":"Microsoft","url":"https://en.wikipedia.org/wiki/Microsoft","dataSource":"Wikipedia"},{"bingId":"0d47c987-0042-5576-15e8-97af601614fa","name":"Bill Gates","matches":[{"text":"Bill Gates","offset":25,"length":10,"confidenceScore":0.52}],"language":"en","id":"Bill Gates","url":"https://en.wikipedia.org/wiki/Bill_Gates","dataSource":"Wikipedia"},{"bingId":"df2c4376-9923-6a54-893f-2ee5a5badbc7","name":"Paul Allen","matches":[{"text":"Paul Allen","offset":40,"length":10,"confidenceScore":0.54}],"language":"en","id":"Paul Allen","url":"https://en.wikipedia.org/wiki/Paul_Allen","dataSource":"Wikipedia"},{"bingId":"52535f87-235e-b513-54fe-c03e4233ac6e","name":"April 4","matches":[{"text":"April 4","offset":54,"length":7,"confidenceScore":0.38}],"language":"en","id":"April 4","url":"https://en.wikipedia.org/wiki/April_4","dataSource":"Wikipedia"}],"warnings":[]}],"errors":[],"modelVersion":"2021-06-01"}}`

type reqBody struct {
	Kind       string `json:"kind"`
	Parameters struct {
		ModelVersion string `json:"modelVersion"`
	} `json:"parameters"`
	AnalysisInput struct {
		Documents []struct {
			ID       string `json:"id"`
			Language string `json:"language"`
			Text     string `json:"text"`
		} `json:"documents"`
	} `json:"analysisInput"`
}

func TestLinkPostsDocumentsAndDecodesEntities(t *testing.T) {
	var path, version, key, ctype string
	var body reqBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, version = r.URL.Path, r.URL.Query().Get("api-version")
		key, ctype = r.Header.Get("Ocp-Apim-Subscription-Key"), r.Header.Get("Content-Type")
		json.NewDecoder(r.Body).Decode(&body)
		w.Write([]byte(sample))
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL + "/", Key: "secret"}
	res, err := c.Link(context.Background(), "Microsoft was founded by Bill Gates and Paul Allen on April 4.", "en")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/language/:analyze-text" || version != APIVersion || key != "secret" || ctype != "application/json" {
		t.Errorf("path=%q version=%q key=%q content-type=%q", path, version, key, ctype)
	}
	docs := body.AnalysisInput.Documents
	if body.Kind != "EntityLinking" || body.Parameters.ModelVersion != "latest" || len(docs) != 1 || docs[0].ID != "1" || docs[0].Language != "en" {
		t.Errorf("body = %+v", body)
	}
	if len(res.Entities) != 4 || res.Entities[0].BingID != "a093e9b9-90f5-a3d5-c4b8-5855e1b01f85" ||
		res.Entities[0].URL != "https://en.wikipedia.org/wiki/Microsoft" || res.Entities[0].Name != "Microsoft" ||
		res.Entities[0].DataSource != "Wikipedia" || res.Entities[0].Language != "en" {
		t.Errorf("entities = %+v", res.Entities)
	}
	if res.Records != 1 {
		t.Errorf("records = %d, want 1", res.Records)
	}
}

func TestDocumentsSplitsAtLineBreaks(t *testing.T) {
	para := func(n int) string { return strings.Repeat("a", n) }
	if got := Documents(para(3000) + "\n" + para(3000) + "\n" + para(3000)); len(got) != 3 {
		t.Errorf("3 x 3000 runes gave %d documents, want 3", len(got))
	}
	if got := Documents(para(2000) + "\n" + para(2000)); len(got) != 1 {
		t.Errorf("2 x 2000 runes gave %d documents, want 1", len(got))
	}
	line := strings.TrimSpace(strings.Repeat("wörd ab ", 1500)) // 11,999 runes
	docs := Documents(line)
	if len(docs) != 3 {
		t.Fatalf("12000-rune line gave %d documents, want 3", len(docs))
	}
	for i, d := range docs {
		if n := utf8.RuneCountInString(d); n > MaxDocChars {
			t.Errorf("doc %d has %d runes", i, n)
		}
	}
	if strings.Join(strings.Fields(strings.Join(docs, " ")), " ") != strings.Join(strings.Fields(line), " ") {
		t.Errorf("words changed by splitting")
	}
}

func TestLinkSendsAtMostFiveDocumentsPerRequest(t *testing.T) {
	var sizes []int
	var ids []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b reqBody
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &b)
		sizes = append(sizes, len(b.AnalysisInput.Documents))
		for _, d := range b.AnalysisInput.Documents {
			ids = append(ids, d.ID)
		}
		w.Write([]byte(sample))
	}))
	defer srv.Close()
	paras := make([]string, 7)
	for i := range paras {
		paras[i] = strings.Repeat("a", 3000)
	}
	res, err := (&Client{Endpoint: srv.URL, Key: "k"}).Link(context.Background(), strings.Join(paras, "\n"), "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 2 || sizes[0] != 5 || sizes[1] != 2 {
		t.Errorf("request sizes = %v, want [5 2]", sizes)
	}
	if strings.Join(ids, ",") != "1,2,3,4,5,6,7" {
		t.Errorf("ids = %v", ids)
	}
	if len(res.Entities) != 8 {
		t.Errorf("entities = %d, want 8 merged", len(res.Entities))
	}
	if res.Records != 21 {
		t.Errorf("records = %d, want 21", res.Records)
	}
}

func TestRecordsCountsThousandCharacterBlocks(t *testing.T) {
	for n, want := range map[int]int{1: 1, 1000: 1, 1001: 2, 5120: 6} {
		if got := Records([]string{strings.Repeat("é", n)}); got != want {
			t.Errorf("Records(%d runes) = %d, want %d", n, got, want)
		}
	}
}

func TestLinkErrorNeverEchoesTheKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Access denied due to invalid subscription key.", http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := (&Client{Endpoint: srv.URL, Key: "secret-key-123"}).Link(context.Background(), "text", "en")
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid subscription") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "secret-key-123") || strings.Contains(err.Error(), "api-version") {
		t.Fatalf("error leaks the key or query: %v", err)
	}
	// A transport failure must not leak the key either.
	srv.Close()
	_, err = (&Client{Endpoint: srv.URL, Key: "secret-key-123"}).Link(context.Background(), "text", "en")
	if err == nil || strings.Contains(err.Error(), "secret-key-123") {
		t.Fatalf("transport err = %v", err)
	}
}

func TestLinkReportsDocumentErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"kind":"EntityLinkingResults","results":{"documents":[],"errors":[{"id":"1","error":{"code":"InvalidArgument","message":"Invalid document in request."}}],"modelVersion":"2021-06-01"}}`))
	}))
	defer srv.Close()
	_, err := (&Client{Endpoint: srv.URL, Key: "k"}).Link(context.Background(), "text", "en")
	if err == nil || !strings.Contains(err.Error(), "document 1") || !strings.Contains(err.Error(), "InvalidArgument") {
		t.Fatalf("err = %v", err)
	}
}

func TestSupportedLanguages(t *testing.T) {
	for lang, want := range map[string]bool{"en": true, "es": true, "de": false, "": false} {
		if got := Supported(lang); got != want {
			t.Errorf("Supported(%q) = %v, want %v", lang, got, want)
		}
	}
}
