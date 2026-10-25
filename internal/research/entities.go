package research

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/browser"
	"github.com/dazamate/go-json-schema/schema"
)

// EntityQuery is a list of phrases to look up as Wikipedia/Wikidata
// entities.
type EntityQuery struct {
	Phrases  []string
	Language string // Wikipedia edition, e.g. en; default en
}

// EntityReport holds one entity per distinct phrase, in input order.
type EntityReport struct {
	Language string   `json:"language"`
	Entities []Entity `json:"entities"`
}

// Match statuses.
const (
	StatusExact     = "exact"     // the phrase is an article title
	StatusRedirect  = "redirect"  // the phrase redirects to an article
	StatusSection   = "section"   // the phrase redirects to a section of a broader article
	StatusAmbiguous = "ambiguous" // the phrase is a disambiguation page; see Candidates
	StatusMissing   = "missing"   // no such title; Candidates are search hits
)

// Entity is a phrase resolved to its identities.
type Entity struct {
	Phrase      string          `json:"phrase"`
	Status      string          `json:"status"`
	Title       string          `json:"title,omitempty"`
	Section     string          `json:"section,omitempty"`
	Wikipedia   string          `json:"wikipedia,omitempty"`
	QID         string          `json:"qid,omitempty"`
	Label       string          `json:"label,omitempty"`
	Description string          `json:"description,omitempty"`
	Aliases     []string        `json:"aliases,omitempty"`
	FreebaseID  string          `json:"freebase_id,omitempty"`  // Wikidata P646, /m/...
	GoogleKGID  string          `json:"google_kg_id,omitempty"` // Wikidata P2671, /g/...
	KGMID       string          `json:"kgmid,omitempty"`        // FreebaseID, else GoogleKGID
	KGMIDFrom   string          `json:"kgmid_from,omitempty"`   // item the kgmid was taken from, when not QID
	Types       []Ref           `json:"types,omitempty"`        // P31 instance of and P279 subclass of
	Thing       json.RawMessage `json:"thing,omitempty"`        // ready schema.org node for about/mentions
	Candidates  []Candidate     `json:"candidates,omitempty"`
	Cached      bool            `json:"cached,omitempty"`
}

// Ref names a Wikidata item.
type Ref struct {
	QID   string `json:"qid"`
	Label string `json:"label"`
}

// Candidate is an article the phrase may mean.
type Candidate struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	QID         string `json:"qid,omitempty"`
}

const (
	maxPhrases    = 100
	maxCandidates = 25
	maxAliases    = 15
	maxTypes      = 5
	batchSize     = 50 // Wikipedia titles and Wikidata ids per request
)

var languageCode = regexp.MustCompile(`^[a-z]{2,3}(-[a-z]{2,8})?$`)

// ResolveEntities looks each phrase up as a Wikipedia title, following
// redirects, then reads the Wikidata item for its Google IDs (Freebase
// P646 and Google Knowledge Graph P2671), aliases and types. An item with
// no Google ID borrows one from an item it names as the same thing under
// another name (see synonymProps), as with Rosemary, whose 2020 taxon item
// points to the older one holding /m/06cd7. Disambiguation pages and
// unknown phrases come back with candidates to choose from.
func (r *Researcher) ResolveEntities(ctx context.Context, q EntityQuery) (EntityReport, error) {
	lang := strings.ToLower(strings.TrimSpace(q.Language))
	if lang == "" {
		lang = "en"
	}
	if !languageCode.MatchString(lang) {
		return EntityReport{}, fmt.Errorf("bad language %q: want a Wikipedia edition code such as en", q.Language)
	}
	phrases := dedupe(q.Phrases)
	if len(phrases) == 0 {
		return EntityReport{}, errors.New("no phrases given")
	}
	if len(phrases) > maxPhrases {
		return EntityReport{}, fmt.Errorf("%d phrases given; at most %d per call", len(phrases), maxPhrases)
	}
	rep := EntityReport{Language: lang, Entities: make([]Entity, len(phrases))}
	var todo []string
	var todoIdx []int
	for i, p := range phrases {
		var e Entity
		if r.Cache.Get(entityBucket, lang+"|"+strings.ToLower(p), entityTTL, &e) {
			e.Phrase, e.Cached = p, true
			rep.Entities[i] = e
			continue
		}
		todo = append(todo, p)
		todoIdx = append(todoIdx, i)
	}
	if len(todo) == 0 {
		return rep, nil
	}
	err := r.Browser.Use(ctx, func(p browser.Page) error {
		res := &resolver{ctx: ctx, p: p, lang: lang}
		if err := res.open(); err != nil {
			return err
		}
		ents, err := res.resolve(todo)
		if err != nil {
			return err
		}
		for k, e := range ents {
			rep.Entities[todoIdx[k]] = e
			r.Cache.Put(entityBucket, lang+"|"+strings.ToLower(e.Phrase), e)
		}
		return nil
	})
	if err != nil {
		return EntityReport{}, err
	}
	return rep, nil
}

// dedupe trims phrases and drops repeats ignoring case. "|" separates
// titles in API calls, so it is removed.
func dedupe(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range in {
		p = strings.Join(strings.Fields(strings.ReplaceAll(p, "|", " ")), " ")
		if low := strings.ToLower(p); p != "" && !seen[low] {
			seen[low] = true
			out = append(out, p)
		}
	}
	return out
}

type resolver struct {
	ctx  context.Context
	p    browser.Page
	lang string
}

func (r *resolver) origin() string { return "https://" + r.lang + ".wikipedia.org" }

// open puts the tab on the Wikipedia edition, so its API calls are
// same-origin. Wikidata calls use anonymous CORS (origin=*).
func (r *resolver) open() error {
	var origin string
	if err := r.p.Eval(r.ctx, "location.origin", &origin); err == nil && origin == r.origin() {
		return nil
	}
	_, err := r.p.Navigate(r.ctx, r.origin()+"/wiki/Special:BlankPage")
	return err
}

type fetched struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
	Error  string `json:"error,omitempty"`
}

// fetchJS fetches every URL from inside the page, six at a time, and
// returns each status and body.
const fetchJS = `(async (urls) => {
  const out = new Array(urls.length);
  let next = 0;
  const worker = async () => {
    while (next < urls.length) {
      const i = next++;
      try { const r = await fetch(urls[i], {credentials: 'omit'}); out[i] = {status: r.status, body: await r.text()}; }
      catch (e) { out[i] = {status: 0, body: '', error: String(e)}; }
    }
  };
  await Promise.all(Array.from({length: Math.min(6, urls.length)}, worker));
  return out;
})`

// fetch GETs the API URLs through the browser and decodes each JSON body
// into the matching element of outs.
func (r *resolver) fetch(urls []string, outs []any) error {
	if len(urls) == 0 {
		return nil
	}
	var res []fetched
	if err := r.p.Eval(r.ctx, call(fetchJS, urls), &res); err != nil {
		return err
	}
	if len(res) != len(urls) {
		return fmt.Errorf("fetched %d of %d API calls", len(res), len(urls))
	}
	for i, f := range res {
		if f.Status != 200 {
			return fmt.Errorf("GET %s: status %d %s", urls[i], f.Status, f.Error)
		}
		if err := json.Unmarshal([]byte(f.Body), outs[i]); err != nil {
			return fmt.Errorf("GET %s: %w", urls[i], err)
		}
	}
	return nil
}

func (r *resolver) wikiAPI(params url.Values) string {
	params.Set("format", "json")
	params.Set("formatversion", "2")
	params.Set("origin", "*")
	return r.origin() + "/w/api.php?" + params.Encode()
}

type wikiResp struct {
	Query struct {
		Normalized []struct{ From, To string } `json:"normalized"`
		Redirects  []struct {
			From       string `json:"from"`
			To         string `json:"to"`
			Tofragment string `json:"tofragment"`
		} `json:"redirects"`
		Pages  []wikiRespPage `json:"pages"`
		Search []struct {
			Title string `json:"title"`
		} `json:"search"`
	} `json:"query"`
	Continue map[string]string `json:"continue"`
}

type wikiRespPage struct {
	Title string `json:"title"`
	Links []struct {
		Title string `json:"title"`
	} `json:"links"`
	LinksHere []struct {
		Title string `json:"title"`
	} `json:"linkshere"`
	Missing     bool              `json:"missing"`
	Invalid     bool              `json:"invalid"`
	Description string            `json:"description"`
	PageProps   map[string]string `json:"pageprops"`
}

func (p wikiRespPage) disambiguation() bool {
	_, ok := p.PageProps["disambiguation"]
	return ok
}

// lookup is where a title landed.
type lookup struct {
	page       wikiRespPage
	fragment   string
	redirected bool
}

// titles looks titles up, following redirects, keyed by the input title.
// Titles go out sorted, so the same set always makes the same requests.
func (r *resolver) titles(titles []string) (map[string]lookup, error) {
	titles = slices.Sorted(slices.Values(titles))
	var urls []string
	var batches [][]string
	for start := 0; start < len(titles); start += batchSize {
		b := titles[start:min(start+batchSize, len(titles))]
		batches = append(batches, b)
		urls = append(urls, r.wikiAPI(url.Values{
			"action": {"query"}, "redirects": {"1"}, "prop": {"pageprops|description"},
			"ppprop": {"wikibase_item|disambiguation"}, "titles": {strings.Join(b, "|")},
		}))
	}
	resps := make([]wikiResp, len(urls))
	if err := r.fetch(urls, ptrs(resps)); err != nil {
		return nil, err
	}
	out := map[string]lookup{}
	for i, resp := range resps {
		q := resp.Query
		for _, t := range batches[i] {
			n, l := t, lookup{}
			for _, norm := range q.Normalized {
				if norm.From == n {
					n = norm.To
				}
			}
			for _, red := range q.Redirects {
				if red.From == n {
					n, l.fragment, l.redirected = red.To, red.Tofragment, true
				}
			}
			for _, pg := range q.Pages {
				if pg.Title == n {
					l.page = pg
				}
			}
			if l.page.Title == "" {
				l.page = wikiRespPage{Title: n, Missing: true}
			}
			out[t] = l
		}
	}
	return out, nil
}

func ptrs[T any](s []T) []any {
	out := make([]any, len(s))
	for i := range s {
		out[i] = &s[i]
	}
	return out
}

func (r *resolver) resolve(phrases []string) ([]Entity, error) {
	looks, err := r.titles(phrases)
	if err != nil {
		return nil, err
	}
	ents := make([]Entity, len(phrases))
	// Disambiguation pages list their meanings as links; unknown phrases
	// get search hits. Both are fetched in one round.
	var urls []string
	var owners []int
	for i, p := range phrases {
		l := looks[p]
		ents[i].Phrase = p
		switch {
		case l.page.Missing || l.page.Invalid:
			ents[i].Status = StatusMissing
			urls = append(urls, r.wikiAPI(url.Values{"action": {"query"}, "list": {"search"}, "srsearch": {p}, "srlimit": {"5"}, "srnamespace": {"0"}}))
			owners = append(owners, i)
		case l.page.disambiguation():
			ents[i].Status = StatusAmbiguous
			urls = append(urls, r.wikiAPI(url.Values{
				"action": {"query"}, "generator": {"links"}, "titles": {l.page.Title}, "gplnamespace": {"0"}, "gpllimit": {"max"},
				"redirects": {"1"}, "prop": {"pageprops|description"}, "ppprop": {"wikibase_item|disambiguation"},
			}))
			owners = append(owners, i)
		default:
			ents[i].Status = StatusExact
			if l.redirected {
				ents[i].Status = StatusRedirect
			}
			if l.fragment != "" {
				ents[i].Status = StatusSection
				ents[i].Section = l.fragment
			}
			ents[i].Title = l.page.Title
			ents[i].QID = l.page.PageProps["wikibase_item"]
			ents[i].Description = l.page.Description
			article := l.page.Title
			if l.fragment != "" {
				article += "#" + l.fragment
			}
			ents[i].Wikipedia = string(schema.WikipediaLang(r.lang, article))
		}
	}
	resps := make([]wikiResp, len(urls))
	if err := r.fetch(urls, ptrs(resps)); err != nil {
		return nil, err
	}
	var searchTitles []string
	for k, resp := range resps {
		e := &ents[owners[k]]
		if e.Status == StatusAmbiguous {
			for _, pg := range resp.Query.Pages {
				if !pg.Missing && !pg.Invalid && !pg.disambiguation() && len(e.Candidates) < maxCandidates {
					e.Candidates = append(e.Candidates, Candidate{Title: pg.Title, Description: pg.Description, QID: pg.PageProps["wikibase_item"]})
				}
			}
			continue
		}
		for _, hit := range resp.Query.Search {
			e.Candidates = append(e.Candidates, Candidate{Title: hit.Title})
			searchTitles = append(searchTitles, hit.Title)
		}
	}
	if err := r.describe(ents, searchTitles); err != nil {
		return nil, err
	}
	if err := r.wikidata(ents); err != nil {
		return nil, err
	}
	for i := range ents {
		ents[i].Thing = r.thing(ents[i])
	}
	return ents, nil
}

// describe fills in search-hit candidates' descriptions and items,
// dropping hits that are disambiguation pages.
func (r *resolver) describe(ents []Entity, titles []string) error {
	if len(titles) == 0 {
		return nil
	}
	looks, err := r.titles(dedupe(titles))
	if err != nil {
		return err
	}
	for i := range ents {
		if ents[i].Status != StatusMissing {
			continue
		}
		var kept []Candidate
		seen := map[string]bool{}
		for _, c := range ents[i].Candidates {
			l, ok := looks[c.Title]
			if !ok || l.page.Missing || l.page.disambiguation() || seen[l.page.Title] {
				continue
			}
			seen[l.page.Title] = true
			kept = append(kept, Candidate{Title: l.page.Title, Description: l.page.Description, QID: l.page.PageProps["wikibase_item"]})
		}
		ents[i].Candidates = kept
	}
	return nil
}

type wdEntity struct {
	ID           string                              `json:"id"`
	Missing      *string                             `json:"missing"`
	Labels       map[string]struct{ Value string }   `json:"labels"`
	Descriptions map[string]struct{ Value string }   `json:"descriptions"`
	Aliases      map[string][]struct{ Value string } `json:"aliases"`
	Sitelinks    map[string]struct{ Title string }   `json:"sitelinks"`
	Claims       map[string][]struct {
		Mainsnak struct {
			Datavalue struct {
				Value json.RawMessage `json:"value"`
			} `json:"datavalue"`
		} `json:"mainsnak"`
		Rank string `json:"rank"`
	} `json:"claims"`
}

// values returns a claim's string values, or item IDs for item claims.
func (e wdEntity) values(prop string) []string {
	var out []string
	for _, c := range e.Claims[prop] {
		if c.Rank == "deprecated" {
			continue
		}
		raw := c.Mainsnak.Datavalue.Value
		var s string
		if json.Unmarshal(raw, &s) == nil {
			out = append(out, s)
			continue
		}
		var item struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &item) == nil && item.ID != "" {
			out = append(out, item.ID)
		}
	}
	return out
}

func (r *resolver) text(m map[string]struct{ Value string }) string {
	if v, ok := m[r.lang]; ok {
		return v.Value
	}
	return m["en"].Value
}

// items fetches Wikidata items by ID.
func (r *resolver) items(ids []string, props string) (map[string]wdEntity, error) {
	ids = dedupe(ids)
	langs := r.lang
	if r.lang != "en" {
		langs += "|en"
	}
	var urls []string
	for start := 0; start < len(ids); start += batchSize {
		b := ids[start:min(start+batchSize, len(ids))]
		urls = append(urls, "https://www.wikidata.org/w/api.php?"+url.Values{
			"action": {"wbgetentities"}, "format": {"json"}, "origin": {"*"}, "ids": {strings.Join(b, "|")},
			"props": {props}, "languages": {langs},
		}.Encode())
	}
	resps := make([]struct {
		Entities map[string]wdEntity `json:"entities"`
	}, len(urls))
	if err := r.fetch(urls, ptrs(resps)); err != nil {
		return nil, err
	}
	out := map[string]wdEntity{}
	for _, resp := range resps {
		for id, e := range resp.Entities {
			if e.Missing == nil {
				out[id] = e
			}
		}
	}
	return out, nil
}

// wikidata fills in labels, aliases, Google IDs and types from Wikidata.
func (r *resolver) wikidata(ents []Entity) error {
	var ids []string
	for _, e := range ents {
		if e.QID != "" {
			ids = append(ids, e.QID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	items, err := r.items(ids, "labels|descriptions|aliases|claims")
	if err != nil {
		return err
	}
	// Second round: type labels, and synonym items for a missing Google ID.
	var more []string
	for i := range ents {
		it, ok := items[ents[i].QID]
		if !ok {
			continue
		}
		e := &ents[i]
		e.Label = r.text(it.Labels)
		if e.Description == "" {
			e.Description = r.text(it.Descriptions)
		}
		aliases := it.Aliases[r.lang]
		for _, a := range aliases[:min(len(aliases), maxAliases)] {
			e.Aliases = append(e.Aliases, a.Value)
		}
		e.FreebaseID = first(it.values("P646"))
		e.GoogleKGID = first(it.values("P2671"))
		e.KGMID = orElse(e.FreebaseID, e.GoogleKGID)
		for _, p := range []string{"P31", "P279"} {
			vals := it.values(p)
			for _, q := range vals[:min(len(vals), maxTypes)] {
				e.Types = append(e.Types, Ref{QID: q})
				more = append(more, q)
			}
		}
		if e.KGMID == "" {
			more = append(more, synonyms(it)...)
		}
	}
	if len(more) == 0 {
		return nil
	}
	extra, err := r.items(more, "labels|claims")
	if err != nil {
		return err
	}
	for i := range ents {
		e := &ents[i]
		for k := range e.Types {
			e.Types[k].Label = r.text(extra[e.Types[k].QID].Labels)
		}
		if e.KGMID != "" || e.QID == "" {
			continue
		}
		it := items[e.QID]
		for _, syn := range synonyms(it) {
			if s, ok := extra[syn]; ok {
				if id := orElse(first(s.values("P646")), first(s.values("P2671"))); id != "" {
					e.KGMID, e.KGMIDFrom = id, syn
					break
				}
			}
		}
	}
	return nil
}

// synonymProps link an item to another item for the same thing: replaced
// synonym, basionym, original combination, taxon synonym, and said to be
// the same as.
var synonymProps = []string{"P694", "P566", "P1403", "P1420", "P460"}

func synonyms(it wdEntity) []string {
	var out []string
	for _, p := range synonymProps {
		out = append(out, it.values(p)...)
	}
	return out
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func orElse(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// thing builds the entity's schema.org node with every sameAs URL and its
// GoogleMachineID, named after the section when the phrase is one.
func (r *resolver) thing(e Entity) json.RawMessage {
	if e.QID == "" && e.Title == "" {
		return nil
	}
	se := schema.Entity{Name: orElse(e.Section, e.Title), Wikidata: e.QID, KGMID: e.KGMID}
	if r.lang == "en" {
		article := e.Title
		if e.Section != "" {
			article += "#" + e.Section
		}
		se.Wikipedia, se.DBpedia = article, e.Title
	} else {
		se.SameAs = []string{e.Wikipedia}
	}
	b, err := schema.Marshal(se.Thing())
	if err != nil {
		return nil
	}
	return b
}
