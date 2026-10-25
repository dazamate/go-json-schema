package research

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dazamate/go-json-schema/internal/browser"
)

// TestRecordSelectFixtures records testdata/select/wiki.json: it runs
// select_entities on the calibration page with the recorded TextRazor answer
// and cosines, but with the real Wikipedia and Wikidata through ChromiumFish.
// It needs chromiumfish installed:
//
//	SCHEMA_MCP_RECORD=1 go test ./internal/research -run RecordSelect -v -timeout 10m
func TestRecordSelectFixtures(t *testing.T) {
	if os.Getenv("SCHEMA_MCP_RECORD") == "" {
		t.Skip("set SCHEMA_MCP_RECORD=1 to re-record testdata/select/wiki.json")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	b := browser.New("go-json-schema-record")
	defer b.Close()
	rec := &wikiRecorder{t: t, b: b, raw: calibrationPage(t, calibrationURL), fx: wikiFixture{
		Items: map[string]json.RawMessage{}, Links: map[string][]string{}, Options: map[string][]string{},
		Pages: map[string]fixturePage{}, Redirects: map[string]string{},
	}}
	rep, err := calibrationResearcher(t, rec).SelectEntities(ctx, calibrationQuery())
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.MarshalIndent(rec.fx, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(selectFixturePath("wiki.json"), append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	links := 0
	for _, l := range rec.fx.Links {
		links += len(l)
	}
	t.Logf("wrote wiki.json: %d items, %d pages, %d redirects, %d option lists, %d pages with %d links, %d bytes",
		len(rec.fx.Items), len(rec.fx.Pages), len(rec.fx.Redirects), len(rec.fx.Options), len(rec.fx.Links), links, len(out)+1)
	for title, opts := range rec.fx.Options {
		t.Logf("options of %s: %d", title, len(opts))
	}
	t.Logf("about: %v", selNames(rep.About))
	t.Logf("mentions: %v", selNames(rep.Mentions))
	t.Logf("note: %s", rep.Note)
}

// wikiRecorder is the real browser, except that our page is answered from
// the fixtures without loading it. It takes every fetchJS call apart into
// the wiki.json maps.
type wikiRecorder struct {
	t   *testing.T
	b   Browser
	raw rawPage
	fx  wikiFixture
}

func (r *wikiRecorder) Use(ctx context.Context, fn func(browser.Page) error) error {
	return r.b.Use(ctx, func(p browser.Page) error { return fn(recordPage{r, p}) })
}

type recordPage struct {
	r *wikiRecorder
	p browser.Page
}

func (p recordPage) Navigate(ctx context.Context, u string) (string, error) {
	if u == calibrationURL {
		return u, nil
	}
	return p.p.Navigate(ctx, u)
}

func (p recordPage) Eval(ctx context.Context, expr string, out any) error {
	if expr == pageJS {
		raw := p.r.raw
		raw.Href = calibrationURL
		return roundTrip(raw, out)
	}
	args, ok := strings.CutPrefix(expr, fetchJS+"(")
	if !ok {
		return p.p.Eval(ctx, expr, out)
	}
	var urls []string
	if err := json.Unmarshal([]byte(strings.TrimSuffix(args, ")")), &urls); err != nil {
		return err
	}
	var res []fetched
	if err := p.p.Eval(ctx, expr, &res); err != nil {
		return err
	}
	for i, f := range res {
		if i < len(urls) && f.Status == 200 {
			p.r.add(urls[i], f.Body)
		}
	}
	return roundTrip(res, out)
}

func roundTrip(v, out any) error {
	if out == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// add files one API answer under its QIDs and titles. An answer the fake
// Wikipedia could not replay fails the recording.
func (r *wikiRecorder) add(raw, body string) {
	t := r.t
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := u.Query()
	if u.Host == "www.wikidata.org" && v.Get("action") == "wbgetentities" {
		var resp struct {
			Entities map[string]json.RawMessage `json:"entities"`
		}
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("GET %s: %v", raw, err)
		}
		for id, e := range resp.Entities {
			r.fx.Items[id] = sortedJSON(t, e)
		}
		return
	}
	if u.Host != "en.wikipedia.org" {
		t.Fatalf("unexpected API call %s", raw)
	}
	var resp wikiResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("GET %s: %v", raw, err)
	}
	for _, n := range resp.Query.Normalized {
		if n.To != ucfirst(n.From) {
			t.Fatalf("GET %s: %q is normalised to %q, which the fake can't replay", raw, n.From, n.To)
		}
	}
	for _, red := range resp.Query.Redirects {
		r.fx.Redirects[red.From] = red.To
	}
	switch {
	case v.Get("prop") == "links":
		for _, pg := range resp.Query.Pages {
			for _, l := range pg.Links {
				addUnique(r.fx.Links, pg.Title, l.Title)
			}
		}
	case v.Get("generator") == "links":
		for _, pg := range resp.Query.Pages {
			r.addPage(pg)
			addUnique(r.fx.Options, v.Get("titles"), pg.Title)
		}
	case v.Get("titles") != "" && strings.Contains(v.Get("prop"), "pageprops"):
		for _, pg := range resp.Query.Pages {
			r.addPage(pg)
		}
	default:
		t.Fatalf("unexpected API call %s", raw)
	}
}

// addPage keeps what a title lookup found. A page repeated by a continue
// never loses its QID.
func (r *wikiRecorder) addPage(pg wikiRespPage) {
	old := r.fx.Pages[pg.Title]
	p := fixturePage{QID: pg.PageProps["wikibase_item"], Disambig: pg.disambiguation(), Missing: pg.Missing || pg.Invalid}
	if p.QID == "" {
		p.QID = old.QID
	}
	p.Disambig = p.Disambig || old.Disambig
	r.fx.Pages[pg.Title] = p
}

func addUnique(m map[string][]string, key, v string) {
	if !slices.Contains(m[key], v) {
		m[key] = append(m[key], v)
	}
}

// sortedJSON re-encodes raw with its object keys sorted, keeping numbers as
// written.
func sortedJSON(t *testing.T, raw json.RawMessage) json.RawMessage {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
