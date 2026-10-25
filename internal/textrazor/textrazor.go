// Package textrazor is a minimal client for the TextRazor entity-linking
// API (https://www.textrazor.com/docs/rest). It only ever sends text: the
// url parameter, which makes TextRazor fetch pages itself, is never used,
// so every page is still fetched through ChromiumFish.
package textrazor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// DefaultEndpoint is TextRazor's secure analysis endpoint.
const DefaultEndpoint = "https://api.textrazor.com/"

// UnitBytes is the text size TextRazor bills as one request.
const UnitBytes = 10 * 1024

// Client calls the analysis endpoint with an API key.
type Client struct {
	Key      string
	Endpoint string       // default DefaultEndpoint
	HTTP     *http.Client // default: a client with a 90 s timeout
}

// Entity is one mention of a linked entity. TextRazor returns one per
// mention, so an entity used three times appears three times.
type Entity struct {
	ID          string   `json:"entityId"`        // Wikipedia title in the text's language
	EnglishID   string   `json:"entityEnglishId"` // English Wikipedia title, when linked
	WikiLink    string   `json:"wikiLink"`
	WikidataID  string   `json:"wikidataId"`
	FreebaseID  string   `json:"freebaseId"`
	MatchedText string   `json:"matchedText"`
	Relevance   float64  `json:"relevanceScore"`  // 0 to 1: how on-topic for the text
	Confidence  float64  `json:"confidenceScore"` // about 0.5 to 10: how sure the link is right
	Types       []string `json:"type"`            // DBpedia types
}

// Topic is a theme of the whole text.
type Topic struct {
	Label      string  `json:"label"`
	WikiLink   string  `json:"wikiLink"`
	WikidataID string  `json:"wikidataId"`
	Score      float64 `json:"score"` // 0 to 1
}

// Result is what one analysis found.
type Result struct {
	Language string   `json:"language"`
	Entities []Entity `json:"entities"`
	Topics   []Topic  `json:"topics"`
	Units    int      `json:"units"` // billed requests: one per 10 KB of text
}

type response struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error"`
	Response struct {
		Language string   `json:"language"`
		Entities []Entity `json:"entities"`
		Topics   []Topic  `json:"topics"`
	} `json:"response"`
}

// Analyze extracts the entities and topics in text. TextRazor accepts at
// most 200 KB per request; Clip text first to control the units billed.
func (c *Client) Analyze(ctx context.Context, text string) (Result, error) {
	form := url.Values{"text": {text}, "extractors": {"entities,topics"}, "cleanup.mode": {"raw"}}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-TextRazor-Key", c.Key)
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 90 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		// url.Error holds only the method and URL, never the key header.
		return Result{}, fmt.Errorf("textrazor: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return Result{}, fmt.Errorf("textrazor: reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.Join(strings.Fields(string(body)), " ")
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return Result{}, fmt.Errorf("textrazor: HTTP %d: %s", resp.StatusCode, msg)
	}
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return Result{}, fmt.Errorf("textrazor: decoding response: %w", err)
	}
	if !r.OK {
		return Result{}, fmt.Errorf("textrazor: %s", orElse(r.Error, "request failed"))
	}
	return Result{Language: r.Response.Language, Entities: r.Response.Entities, Topics: r.Response.Topics, Units: Units(len(text))}, nil
}

// Units is how many requests TextRazor bills for n bytes of text.
func Units(n int) int {
	if n <= UnitBytes {
		return 1
	}
	return (n + UnitBytes - 1) / UnitBytes
}

// Clip shortens text to at most max bytes, cutting at the last space
// before the limit so no word or UTF-8 character is split.
func Clip(text string, max int) string {
	if len(text) <= max {
		return text
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	if sp := strings.LastIndexAny(text[:cut], " \n\t"); sp > 0 {
		cut = sp
	}
	return strings.TrimRight(text[:cut], " \n\t")
}

func orElse(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
