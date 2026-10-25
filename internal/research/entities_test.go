package research

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

type wikiPage struct {
	qid, desc string
	disambig  bool
	links     []string
}

type wikiItem struct {
	label, desc string
	aliases     []string
	claims      map[string][]string // P646 -> ["/m/.."], P31 -> ["Q5"]
	sitelink    string              // English Wikipedia title, emitted as sitelinks.enwiki
}

// fakeWiki answers the Wikipedia and Wikidata API calls that resolve
// makes, from in-memory pages and items.
type fakeWiki struct {
	pages     map[string]wikiPage
	redirects map[string][2]string // from -> {to, fragment}
	search    map[string][]string
	items     map[string]wikiItem
	fetches   int

	articleLinks     map[string][]string        // page -> pages it links to, answered for prop=links
	linksPerResponse int                        // when above 0, prop=links answers are cut here with a continue
	genPerResponse   int                        // the same for generator=links answers
	rawItems         map[string]json.RawMessage // returned as is by wbgetentities
}

// linksJSON answers prop=links, listing for each requested page only the
// links that are among pltitles.
func (w *fakeWiki) linksJSON(v url.Values) map[string]any {
	want := map[string]bool{}
	for _, t := range strings.Split(v.Get("pltitles"), "|") {
		want[t] = true
	}
	type pair struct{ from, to string }
	var order []string
	var pairs []pair
	for _, t := range strings.Split(v.Get("titles"), "|") {
		n := ucfirst(t)
		if r, ok := w.redirects[n]; ok && v.Get("redirects") == "1" {
			n = r[0]
		}
		order = append(order, n)
		for _, l := range w.articleLinks[n] {
			if want[l] {
				pairs = append(pairs, pair{n, l})
			}
		}
	}
	start, _ := strconv.Atoi(v.Get("plcontinue"))
	end, more := len(pairs), false
	if w.linksPerResponse > 0 && start+w.linksPerResponse < end {
		end, more = start+w.linksPerResponse, true
	}
	var pages []map[string]any
	for _, n := range order {
		pg := map[string]any{"ns": 0, "title": n}
		var links []map[string]any
		for _, p := range pairs[start:end] {
			if p.from == n {
				links = append(links, map[string]any{"ns": 0, "title": p.to})
			}
		}
		if links != nil {
			pg["links"] = links
		}
		pages = append(pages, pg)
	}
	out := map[string]any{"query": map[string]any{"pages": pages}}
	if more {
		out["continue"] = map[string]any{"plcontinue": strconv.Itoa(end), "continue": "||"}
	}
	return out
}

func ucfirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

func (w *fakeWiki) pagesJSON(titles []string, followRedirects bool) map[string]any {
	q := map[string]any{}
	var normalized, redirects, pages []map[string]any
	for _, t := range titles {
		n := ucfirst(t)
		if n != t {
			normalized = append(normalized, map[string]any{"from": t, "to": n})
		}
		if r, ok := w.redirects[n]; ok && followRedirects {
			red := map[string]any{"from": n, "to": r[0]}
			if r[1] != "" {
				red["tofragment"] = r[1]
			}
			redirects = append(redirects, red)
			n = r[0]
		}
		p, ok := w.pages[n]
		if !ok {
			pages = append(pages, map[string]any{"ns": 0, "title": n, "missing": true})
			continue
		}
		props := map[string]any{"wikibase_item": p.qid}
		if p.disambig {
			props["disambiguation"] = ""
		}
		pages = append(pages, map[string]any{"ns": 0, "title": n, "description": p.desc, "pageprops": props})
	}
	q["normalized"], q["redirects"], q["pages"] = normalized, redirects, pages
	return map[string]any{"query": q}
}

func (w *fakeWiki) answer(raw string) (any, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	v := u.Query()
	if v.Get("origin") != "*" || v.Get("format") != "json" {
		return nil, fmt.Errorf("bad API call %s", raw)
	}
	switch {
	case u.Host == "www.wikidata.org" && v.Get("action") == "wbgetentities":
		ents := map[string]any{}
		for _, id := range strings.Split(v.Get("ids"), "|") {
			if raw, ok := w.rawItems[id]; ok {
				ents[id] = raw
				continue
			}
			it, ok := w.items[id]
			if !ok {
				ents[id] = map[string]any{"id": id, "missing": ""}
				continue
			}
			claims := map[string]any{}
			for p, vals := range it.claims {
				var list []any
				for _, val := range vals {
					var dv any = val
					if strings.HasPrefix(val, "Q") {
						dv = map[string]any{"entity-type": "item", "id": val}
					}
					list = append(list, map[string]any{"mainsnak": map[string]any{"snaktype": "value", "datavalue": map[string]any{"value": dv}}, "rank": "normal"})
				}
				claims[p] = list
			}
			var aliases []any
			for _, a := range it.aliases {
				aliases = append(aliases, map[string]any{"language": "en", "value": a})
			}
			ent := map[string]any{
				"id":           id,
				"labels":       map[string]any{"en": map[string]any{"value": it.label}},
				"descriptions": map[string]any{"en": map[string]any{"value": it.desc}},
				"aliases":      map[string]any{"en": aliases},
				"claims":       claims,
			}
			if it.sitelink != "" {
				ent["sitelinks"] = map[string]any{"enwiki": map[string]any{"site": "enwiki", "title": it.sitelink}}
			}
			ents[id] = ent
		}
		return map[string]any{"entities": ents}, nil
	case u.Host == "en.wikipedia.org" && v.Get("prop") == "links":
		return w.linksJSON(v), nil
	case u.Host == "en.wikipedia.org" && v.Get("list") == "search":
		var hits []any
		for _, t := range w.search[v.Get("srsearch")] {
			hits = append(hits, map[string]any{"title": t})
		}
		return map[string]any{"query": map[string]any{"search": hits}}, nil
	case u.Host == "en.wikipedia.org" && v.Get("generator") == "links":
		links := w.pages[v.Get("titles")].links
		start, _ := strconv.Atoi(v.Get("gplcontinue"))
		end := len(links)
		if w.genPerResponse > 0 && start+w.genPerResponse < end {
			end = start + w.genPerResponse
		}
		out := w.pagesJSON(links[start:end], true)
		if end < len(links) {
			out["continue"] = map[string]any{"gplcontinue": strconv.Itoa(end), "continue": "gplcontinue||"}
		}
		return out, nil
	case u.Host == "en.wikipedia.org" && v.Get("titles") != "":
		return w.pagesJSON(strings.Split(v.Get("titles"), "|"), v.Get("redirects") == "1"), nil
	}
	return nil, fmt.Errorf("unexpected API call %s", raw)
}

func (w *fakeWiki) page() *fakePage {
	return &fakePage{eval: func(current, expr string) (any, error) {
		if expr == "location.origin" {
			u, _ := url.Parse(current)
			if u == nil || u.Host == "" {
				return "null", nil
			}
			return u.Scheme + "://" + u.Host, nil
		}
		args, ok := strings.CutPrefix(expr, fetchJS+"(")
		if !ok {
			return nil, fmt.Errorf("unexpected eval %.60s", expr)
		}
		var urls []string
		if err := json.Unmarshal([]byte(strings.TrimSuffix(args, ")")), &urls); err != nil {
			return nil, err
		}
		var out []fetched
		for _, raw := range urls {
			w.fetches++
			body, err := w.answer(raw)
			if err != nil {
				return nil, err
			}
			b, _ := json.Marshal(body)
			out = append(out, fetched{Status: 200, Body: string(b)})
		}
		return out, nil
	}}
}

func newFakeWiki() *fakeWiki {
	return &fakeWiki{
		pages: map[string]wikiPage{
			"Lemon":             {qid: "Q500", desc: "Species of citrus fruit"},
			"Chicken as food":   {qid: "Q864693", desc: "Meat of the chicken"},
			"Convection oven":   {qid: "Q951722", desc: "Oven with fans"},
			"Mercury":           {qid: "Q48397", disambig: true, links: []string{"Mercury (planet)", "Mercury (element)", "Mercury"}},
			"Mercury (planet)":  {qid: "Q308", desc: "Planet"},
			"Mercury (element)": {qid: "Q925", desc: "Chemical element"},
			"Rosemary":          {qid: "Q102240169", desc: "Species of plant"},
			"Lemon chicken":     {qid: "Q6522447", desc: "Chicken dish"},
			"Chicken (food)":    {qid: "Q864693"},
		},
		redirects: map[string][2]string{
			"Chicken thigh": {"Chicken as food", ""},
			"Air fryer":     {"Convection oven", "Air fryer"},
		},
		search: map[string][]string{"lemon herb chicken": {"Lemon chicken", "Chicken (food)"}},
		items: map[string]wikiItem{
			"Q500":       {label: "Citrus × limon", desc: "species of plant", aliases: []string{"lemon tree"}, claims: map[string][]string{"P646": {"/m/09k_b"}, "P2671": {"/g/1yxkhgd6l"}, "P31": {"Q16521"}}},
			"Q16521":     {label: "taxon"},
			"Q864693":    {label: "chicken as food", claims: map[string][]string{"P646": {"/m/0f25w9"}, "P279": {"Q10990"}}},
			"Q10990":     {label: "poultry"},
			"Q951722":    {label: "convection oven", claims: map[string][]string{"P646": {"/m/02z5bb"}}},
			"Q102240169": {label: "Salvia rosmarinus", claims: map[string][]string{"P694": {"Q122679"}}},
			"Q122679":    {label: "Rosmarinus officinalis", claims: map[string][]string{"P646": {"/m/06cd7"}}},
		},
	}
}

func resolve(t *testing.T, w *fakeWiki, r *Researcher, phrases ...string) map[string]Entity {
	t.Helper()
	if r == nil {
		r = &Researcher{}
	}
	r.Browser = fakeBrowser{w.page()}
	rep, err := r.ResolveEntities(context.Background(), EntityQuery{Phrases: phrases})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Entity{}
	for _, e := range rep.Entities {
		out[e.Phrase] = e
	}
	return out
}

func TestResolveExactMatchCarriesGoogleIDsAndTypes(t *testing.T) {
	e := resolve(t, newFakeWiki(), nil, "lemon")["lemon"]
	if e.Status != "exact" || e.Title != "Lemon" || e.QID != "Q500" || e.Wikipedia != "https://en.wikipedia.org/wiki/Lemon" {
		t.Fatalf("lemon = %+v", e)
	}
	if e.FreebaseID != "/m/09k_b" || e.GoogleKGID != "/g/1yxkhgd6l" || e.KGMID != "/m/09k_b" {
		t.Errorf("ids = %+v", e)
	}
	if e.Label != "Citrus × limon" || !slices.Equal(e.Aliases, []string{"lemon tree"}) || e.Description != "Species of citrus fruit" {
		t.Errorf("labels = %+v", e)
	}
	if len(e.Types) != 1 || e.Types[0] != (Ref{QID: "Q16521", Label: "taxon"}) {
		t.Errorf("types = %+v", e.Types)
	}
	var thing map[string]any
	if err := json.Unmarshal(e.Thing, &thing); err != nil {
		t.Fatalf("thing: %v %s", err, e.Thing)
	}
	same := fmt.Sprint(thing["sameAs"])
	for _, want := range []string{"https://en.wikipedia.org/wiki/Lemon", "https://www.wikidata.org/wiki/Q500", "kgmid=/m/09k_b", "http://dbpedia.org/resource/Lemon"} {
		if !strings.Contains(same, want) {
			t.Errorf("sameAs lacks %s: %s", want, same)
		}
	}
	if thing["name"] != "Lemon" || !strings.Contains(string(e.Thing), `"value":"kg:/m/09k_b"`) {
		t.Errorf("thing = %s", e.Thing)
	}
}

func TestResolveMarksRedirectsAndSections(t *testing.T) {
	got := resolve(t, newFakeWiki(), nil, "chicken thigh", "air fryer")
	if e := got["chicken thigh"]; e.Status != "redirect" || e.Title != "Chicken as food" || e.KGMID != "/m/0f25w9" {
		t.Errorf("chicken thigh = %+v", e)
	}
	e := got["air fryer"]
	if e.Status != "section" || e.Title != "Convection oven" || e.Section != "Air fryer" || e.Wikipedia != "https://en.wikipedia.org/wiki/Convection_oven#Air_fryer" {
		t.Errorf("air fryer = %+v", e)
	}
	if !strings.Contains(string(e.Thing), `"name":"Air fryer"`) {
		t.Errorf("section thing should be named after the section: %s", e.Thing)
	}
}

func TestResolveListsCandidatesForAmbiguousAndMissing(t *testing.T) {
	got := resolve(t, newFakeWiki(), nil, "mercury", "lemon herb chicken")
	m := got["mercury"]
	if m.Status != "ambiguous" || len(m.Candidates) != 2 || m.Candidates[0] != (Candidate{Title: "Mercury (planet)", Description: "Planet", QID: "Q308"}) {
		t.Errorf("mercury = %+v", m)
	}
	if m.QID != "" || len(m.Thing) != 0 {
		t.Errorf("ambiguous phrase resolved to the disambiguation page: %+v", m)
	}
	l := got["lemon herb chicken"]
	if l.Status != "missing" || len(l.Candidates) != 2 || l.Candidates[0].QID != "Q6522447" {
		t.Errorf("lemon herb chicken = %+v", l)
	}
}

func TestResolveBorrowsGoogleIDFromSynonymItem(t *testing.T) {
	e := resolve(t, newFakeWiki(), nil, "rosemary")["rosemary"]
	if e.QID != "Q102240169" || e.KGMID != "/m/06cd7" || e.KGMIDFrom != "Q122679" || e.FreebaseID != "" {
		t.Fatalf("rosemary = %+v", e)
	}
}

func TestResolveDedupesAndCaches(t *testing.T) {
	w := newFakeWiki()
	r := &Researcher{Cache: NewCache(t.TempDir())}
	got := resolve(t, w, r, "Lemon", "lemon", " lemon ")
	if len(got) != 1 {
		t.Fatalf("got %d entities, want 1: %v", len(got), got)
	}
	before := w.fetches
	again := resolve(t, w, r, "lemon")
	if w.fetches != before || !again["lemon"].Cached {
		t.Fatalf("second lookup fetched %d more times", w.fetches-before)
	}
}

func TestResolveSkipsOldEntityCacheEntries(t *testing.T) {
	r := &Researcher{Cache: NewCache(t.TempDir())}
	old := Entity{Phrase: "lemon", Status: StatusExact, Title: "Lemon", QID: "Q500",
		Thing: json.RawMessage(`{"@type":"Thing","name":"Lemon","sameAs":["https://www.google.com/search?q=Lemon&kponly&kgmid=/m/09k_b"]}`)}
	if err := r.Cache.Put("entity", "en|lemon", old); err != nil {
		t.Fatal(err)
	}
	e := resolve(t, newFakeWiki(), r, "lemon")["lemon"]
	if e.Cached || strings.Contains(string(e.Thing), "kponly") || !strings.Contains(string(e.Thing), "https://www.google.com/search?kgmid=/m/09k_b") {
		t.Errorf("old cache entry served: cached %v, thing %s", e.Cached, e.Thing)
	}
}

func TestResolveRejectsBadLanguage(t *testing.T) {
	r := &Researcher{Browser: fakeBrowser{newFakeWiki().page()}}
	if _, err := r.ResolveEntities(context.Background(), EntityQuery{Phrases: []string{"x"}, Language: "en.evil.com/"}); err == nil {
		t.Fatal("bad language accepted")
	}
}
