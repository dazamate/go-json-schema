// Package azurelang is a minimal client for the Azure AI Language entity
// linking API
// (https://learn.microsoft.com/azure/ai-services/language-service/entity-linking/quickstart).
// It only ever sends text the browser already fetched, and returns the Bing
// entity IDs (bingId) that link each entity to Wikipedia.
package azurelang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// APIVersion is the analyze-text API version the client targets.
const APIVersion = "2022-05-01"

const (
	// MaxDocChars is the longest document, in characters (runes), Azure accepts.
	MaxDocChars = 5120
	// MaxDocsPerRequest is the most documents one request may carry.
	MaxDocsPerRequest = 5
	// RecordChars is the text size Azure bills as one text record.
	RecordChars = 1000
)

// Client calls the analyze-text endpoint with a resource key.
type Client struct {
	Endpoint string       // the resource endpoint, e.g. https://x.cognitiveservices.azure.com/
	Key      string       // sent as the Ocp-Apim-Subscription-Key header, never in the URL
	HTTP     *http.Client // default: a client with a 90 s timeout
}

// Entity is one linked entity. Azure returns one per entity, with its
// matches inside; only the identifying fields are kept.
type Entity struct {
	BingID     string `json:"bingId"`
	Name       string `json:"name"`
	ID         string `json:"id"` // Wikipedia title in the text's language
	URL        string `json:"url"`
	DataSource string `json:"dataSource"`
	Language   string `json:"language"`
}

// Result is what one Link call found.
type Result struct {
	Entities []Entity
	Records  int // billed text records: one per 1,000 characters of each document
}

type document struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Text     string `json:"text"`
}

type response struct {
	Results struct {
		Documents []struct {
			ID       string   `json:"id"`
			Entities []Entity `json:"entities"`
		} `json:"documents"`
		Errors []struct {
			ID    string `json:"id"`
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"errors"`
	} `json:"results"`
}

// Link finds the entities in text, which must be in language (see
// Supported). The text is split with Documents and sent MaxDocsPerRequest
// documents at a time, one request after another; the entities of every
// request are merged.
func (c *Client) Link(ctx context.Context, text, language string) (Result, error) {
	docs := Documents(text)
	res := Result{Records: Records(docs)}
	for start := 0; start < len(docs); start += MaxDocsPerRequest {
		end := min(start+MaxDocsPerRequest, len(docs))
		batch := make([]document, 0, end-start)
		for i := start; i < end; i++ {
			batch = append(batch, document{ID: fmt.Sprint(i + 1), Language: language, Text: docs[i]})
		}
		ents, err := c.analyze(ctx, batch)
		if err != nil {
			return Result{}, err
		}
		res.Entities = append(res.Entities, ents...)
	}
	return res, nil
}

func (c *Client) analyze(ctx context.Context, docs []document) ([]Entity, error) {
	payload := map[string]any{
		"kind":          "EntityLinking",
		"parameters":    map[string]string{"modelVersion": "latest"},
		"analysisInput": map[string]any{"documents": docs},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(c.Endpoint, "/") + "/language/:analyze-text?api-version=" + APIVersion
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", c.Key)
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 90 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		// url.Error holds only the method and URL, never the key header.
		return nil, fmt.Errorf("azurelang: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("azurelang: reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.Join(strings.Fields(string(body)), " ")
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, fmt.Errorf("azurelang: HTTP %d: %s", resp.StatusCode, msg)
	}
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("azurelang: decoding response: %w", err)
	}
	if len(r.Results.Errors) > 0 {
		e := r.Results.Errors[0]
		return nil, fmt.Errorf("azurelang: document %s: %s: %s", e.ID, e.Error.Code, e.Error.Message)
	}
	var ents []Entity
	for _, d := range r.Results.Documents {
		ents = append(ents, d.Entities...)
	}
	return ents, nil
}

// Documents splits text at line breaks into documents of at most MaxDocChars
// characters (runes); a line longer than that is cut at spaces.
func Documents(text string) []string {
	var docs []string
	var cur []string
	n := 0
	flush := func() {
		if len(cur) > 0 {
			docs = append(docs, strings.Join(cur, "\n"))
			cur, n = nil, 0
		}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		for utf8.RuneCountInString(line) > MaxDocChars {
			flush()
			cut := cutAtSpace(line, MaxDocChars)
			docs = append(docs, strings.TrimSpace(line[:cut]))
			line = strings.TrimSpace(line[cut:])
		}
		if line == "" {
			continue
		}
		if l := utf8.RuneCountInString(line); n > 0 && n+l > MaxDocChars { // n counts each line plus its "\n"
			flush()
		}
		cur = append(cur, line)
		n += utf8.RuneCountInString(line) + 1
	}
	flush()
	return docs
}

// cutAtSpace is the byte index of the last space within the first limit
// runes of s, else the byte index of the limit itself.
func cutAtSpace(s string, limit int) int {
	end, runes := len(s), 0
	for i := range s {
		if runes == limit {
			end = i
			break
		}
		runes++
	}
	if sp := strings.LastIndex(s[:end], " "); sp > 0 {
		return sp
	}
	return end
}

// Records is how many text records Azure bills for docs: one per 1,000
// characters of each document, rounded up.
func Records(docs []string) int {
	n := 0
	for _, d := range docs {
		n += (utf8.RuneCountInString(d) + RecordChars - 1) / RecordChars
	}
	return n
}

// Supported reports whether Azure entity linking accepts language: English
// and Spanish only.
func Supported(language string) bool {
	return language == "en" || language == "es"
}
