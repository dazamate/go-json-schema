// Package research gathers the raw material for entity-stuffed schema:
// search results, the ranking pages' terms and schema, and Wikipedia and
// Wikidata identities. All fetching goes through the ChromiumFish browser;
// the one call outside it sends page text already fetched to an
// EntityLinker (TextRazor).
package research

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dazamate/go-json-schema/internal/azurelang"
	"github.com/dazamate/go-json-schema/internal/browser"
	"github.com/dazamate/go-json-schema/internal/cooldown"
)

// Browser gives one caller at a time the browser tab.
type Browser interface {
	Use(ctx context.Context, fn func(browser.Page) error) error
}

// Researcher runs the research tools.
type Researcher struct {
	Browser Browser
	Cache   *Cache             // nil disables caching
	Google  *cooldown.Cooldown // spaces out requests to Google; nil disables
	// Linker links the ranking pages' entities in page_terms; nil skips it.
	Linker       EntityLinker
	LinkMaxBytes int // text sent to Linker per page; default DefaultLinkMaxBytes
	// Bing adds Bing IDs and Themes scores themes in select_entities; nil skips each.
	Bing   BingLinker
	Themes ThemeScorer
}

// BingLinker links the entities in a text to Bing IDs. azurelang.Client is one.
type BingLinker interface {
	Link(ctx context.Context, text, language string) (azurelang.Result, error)
}

// ThemeScorer scores theme labels and graph texts. embed.Scorer is one;
// research does not import embed.
type ThemeScorer interface {
	Model() string
	Score(ctx context.Context, keyword, page string, labels []string) (kw, pg []float64, err error)
	Similarity(ctx context.Context, keyword string, texts []string) ([]float64, error)
}

const (
	serpTTL = 24 * time.Hour
	// serpBucket changes when SERP gains fields, so older entries are not
	// served without them.
	serpBucket = "serp2"
	pageTTL    = 7 * 24 * time.Hour
	entityTTL  = 30 * 24 * time.Hour
	// entityBucket changes when the cached Thing node changes, as when the
	// Google link became ?kgmid= alone, so older nodes are not served.
	entityBucket = "entity2"
)

// SERPQuery is one keyword search. Every field but Keyword is optional.
type SERPQuery struct {
	Keyword         string
	Country         string            // country searched from (gl), e.g. us, au; default us
	Language        string            // interface language (hl), e.g. en, pt-BR; default en
	Domain          string            // Google domain, e.g. google.com.au; default google.com
	Page            int               // results page, 1 to 10; default 1
	TimeRange       string            // past hour, day, week, month or year (tbs=qdr)
	SafeSearch      bool              // filter explicit results (safe=active)
	ResultsLanguage string            // only pages in this language (lr)
	ResultsCountry  string            // only pages from this country (cr)
	NoAutocorrect   bool              // search the keyword as typed (nfpr=1)
	ShowOmitted     bool              // include results Google hides as similar (filter=0)
	WebOnly         bool              // Google's Web filter: links only, no AI Overview (udm=14)
	Near            string            // place to localize results to, e.g. "Austin, TX" (near), like Cora's near button
	Params          map[string]string // other Google URL parameters, set last; q is reserved
	ResolveURLs     bool              // follow redirect links to get every result's exact URL
}

// SERP is the first page of search results for a keyword.
type SERP struct {
	Keyword         string     `json:"keyword"`
	Engine          string     `json:"engine"`
	SearchURL       string     `json:"search_url"`
	Note            string     `json:"note,omitempty"`
	Cached          bool       `json:"cached,omitempty"`
	Results         []Result   `json:"results"`
	PeopleAlsoAsk   []string   `json:"people_also_ask,omitempty"`
	RelatedSearches []string   `json:"related_searches,omitempty"`
	Autocomplete    []string   `json:"autocomplete,omitempty"`
	BoldTerms       []BoldTerm `json:"bold_terms,omitempty"`
	BoldPassages    []string   `json:"bold_passages,omitempty"`
}

// BoldTerm is a short word or phrase the engine bolded in snippets as a
// match for the keyword: a variant, synonym or entity name. Cora SEO
// Software builds its match words the same way.
type BoldTerm struct {
	Term    string `json:"term"`
	Results int    `json:"results"` // results whose snippet bolds it
}

// Result is one organic search result.
type Result struct {
	Rank       int    `json:"rank"`
	Title      string `json:"title"`
	URL        string `json:"url,omitempty"` // exact destination, when known
	Link       string `json:"link"`          // the result's link; may be a Google redirect
	Domain     string `json:"domain"`
	Breadcrumb string `json:"breadcrumb,omitempty"`
	Snippet    string `json:"snippet,omitempty"`
}

type rawSERP struct {
	Walled  bool        `json:"walled"`
	Results []rawResult `json:"results"`
	PAA     []string    `json:"paa"`
	Related []string    `json:"related"`
	Suggest []string    `json:"suggest"`
}

type rawResult struct {
	Title   string   `json:"title"`
	Href    string   `json:"href"`
	SURL    string   `json:"surl"`
	Cite    string   `json:"cite"`
	Snippet string   `json:"snippet"`
	Bold    []string `json:"bold"` // bolded spans in the snippet
}

type engine struct {
	name string
	url  func(q SERPQuery) string
	js   string
}

var engines = []engine{
	{"google", googleURL, googleJS},
	{"duckduckgo", func(q SERPQuery) string {
		return "https://html.duckduckgo.com/html/?" + url.Values{"q": {q.Keyword}, "kl": {q.Country + "-" + baseLanguage(q.Language)}}.Encode()
	}, ddgJS},
	{"bing", func(q SERPQuery) string {
		return "https://www.bing.com/search?" + url.Values{"q": {q.Keyword}, "cc": {q.Country}, "setlang": {q.Language}}.Encode()
	}, bingJS},
}

// googleJS reads the organic results with their snippets' bolded spans,
// People also ask, related searches and autocomplete suggestions. A result's block is the largest ancestor
// of its link holding only its own h3. Result links are Google redirects;
// only video results expose their destination (data-surl).
const googleJS = `(async (a) => {
  const low = s => (s || '').toLowerCase();
  const bolds = (el, sel) => el ? [...el.querySelectorAll(sel)].map(e => e.innerText.trim()).filter(Boolean) : [];
  const body = low(document.body && document.body.innerText).slice(0, 4000);
  const out = {walled: location.pathname.startsWith('/sorry') || low(document.title).includes('not a robot') || body.includes('unusual traffic'), results: [], paa: [], related: [], suggest: []};
  if (out.walled) return out;
  const root = document.querySelector('#rso') || document.querySelector('#search') || document.body;
  root.querySelectorAll('a h3').forEach(h => {
    const link = h.closest('a');
    let block = link;
    while (block.parentElement && block.parentElement !== root && block.parentElement.querySelectorAll('h3').length === 1) block = block.parentElement;
    const cite = block.querySelector('cite');
    const snip = block.querySelector('.VwiC3b, [data-sncf], [style*="-webkit-line-clamp"]');
    const surlEl = block.matches('[data-surl]') ? block : block.querySelector('[data-surl]');
    out.results.push({title: h.innerText.trim(), href: link.href, surl: surlEl ? surlEl.getAttribute('data-surl') : '',
      cite: cite ? cite.textContent.trim() : '', snippet: snip ? snip.innerText.trim() : '', bold: bolds(snip, 'em, b')});
  });
  document.querySelectorAll('[data-q]').forEach(e => out.paa.push(e.getAttribute('data-q')));
  document.querySelectorAll('#botstuff a[href*="/search?"], #bres a[href*="/search?"]').forEach(e => out.related.push(e.innerText.trim()));
  try {
    const r = await fetch('/complete/search?client=chrome&hl=' + encodeURIComponent(a.hl) + '&gl=' + encodeURIComponent(a.gl) + '&q=' + encodeURIComponent(a.q));
    out.suggest = (await r.json())[1] || [];
  } catch (e) {}
  return out;
})`

const ddgJS = `((a) => {
  const out = {walled: false, results: []};
  document.querySelectorAll('.result').forEach(r => {
    if (r.classList.contains('result--ad')) return;
    const link = r.querySelector('a.result__a');
    if (!link) return;
    let u = link.href;
    const m = u.match(/[?&]uddg=([^&]+)/);
    if (m) { try { u = decodeURIComponent(m[1]); } catch (e) {} }
    const s = r.querySelector('.result__snippet'), c = r.querySelector('.result__url');
    out.results.push({title: link.innerText.trim(), href: u, surl: u, cite: c ? c.innerText.trim() : '', snippet: s ? s.innerText.trim() : '',
      bold: s ? [...s.querySelectorAll('b')].map(e => e.innerText.trim()).filter(Boolean) : []});
  });
  out.walled = out.results.length === 0 && /captcha|unusual|anomaly/i.test((document.body.innerText || '').slice(0, 2000));
  return out;
})`

const bingJS = `((a) => {
  const out = {walled: false, results: [], related: []};
  document.querySelectorAll('#b_results > li.b_algo').forEach(r => {
    const link = r.querySelector('h2 a');
    if (!link) return;
    const s = r.querySelector('.b_caption p') || r.querySelector('p'), c = r.querySelector('cite');
    out.results.push({title: link.innerText.trim(), href: link.href, surl: '', cite: c ? c.innerText.trim() : '', snippet: s ? s.innerText.trim() : '',
      bold: s ? [...s.querySelectorAll('strong, b')].map(e => e.innerText.trim()).filter(Boolean) : []});
  });
  document.querySelectorAll('#brsv3 a, .b_rs a').forEach(e => out.related.push(e.innerText.trim()));
  return out;
})`

// call wraps a JS function literal into an expression applied to args.
func call(fn string, args any) string {
	b, _ := json.Marshal(args)
	return fn + "(" + string(b) + ")"
}

var (
	countryCode    = regexp.MustCompile(`^[a-z]{2}$`)
	googleLanguage = regexp.MustCompile(`^([a-z]{2,3})(?:-([a-z]{2}))?$`)
	googleDomain   = regexp.MustCompile(`^google(\.[a-z]{2,3}){1,2}$`)
	paramName      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,31}$`)
)

const maxPage = 10

// timeRanges maps a time range to its Google qdr code.
var timeRanges = map[string]string{"hour": "h", "day": "d", "week": "w", "month": "m", "year": "y"}

// normalize fills in the defaults and checks every param, so a typo fails
// before any request instead of searching without it.
func (q SERPQuery) normalize() (SERPQuery, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Keyword == "" {
		return q, errors.New("keyword is empty")
	}
	var err error
	if q.Country, err = country(q.Country, "us", "country"); err != nil {
		return q, err
	}
	if q.Language, err = language(q.Language, "en", "language"); err != nil {
		return q, err
	}
	if q.ResultsCountry, err = country(q.ResultsCountry, "", "results_country"); err != nil {
		return q, err
	}
	if q.ResultsLanguage, err = language(q.ResultsLanguage, "", "results_language"); err != nil {
		return q, err
	}
	d := strings.ToLower(strings.TrimSpace(q.Domain))
	d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
	d = strings.TrimPrefix(strings.TrimSuffix(d, "/"), "www.")
	if d == "" {
		d = "google.com"
	}
	if !googleDomain.MatchString(d) {
		return q, fmt.Errorf("google_domain %q is not a Google domain such as google.com.au", q.Domain)
	}
	q.Domain = "www." + d
	if q.Page == 0 {
		q.Page = 1
	}
	if q.Page < 1 || q.Page > maxPage {
		return q, fmt.Errorf("page %d is outside 1-%d", q.Page, maxPage)
	}
	q.Near = strings.Join(strings.Fields(q.Near), " ")
	q.TimeRange = strings.ToLower(strings.TrimSpace(q.TimeRange))
	if _, ok := timeRanges[q.TimeRange]; q.TimeRange != "" && !ok {
		return q, fmt.Errorf("time_range %q is not hour, day, week, month or year", q.TimeRange)
	}
	for k := range q.Params {
		if k == "q" {
			return q, errors.New("params cannot set q; the keyword is the query")
		}
		if !paramName.MatchString(k) {
			return q, fmt.Errorf("params key %q is not a URL parameter name", k)
		}
	}
	return q, nil
}

func country(code, def, name string) (string, error) {
	c := strings.ToLower(strings.TrimSpace(code))
	if c == "" {
		return def, nil
	}
	if !countryCode.MatchString(c) {
		return "", fmt.Errorf("%s %q is not a two-letter country code such as us or au", name, code)
	}
	return c, nil
}

// language normalizes a language code such as pt-br to pt-BR.
func language(code, def, name string) (string, error) {
	l := strings.ToLower(strings.TrimSpace(code))
	if l == "" {
		return def, nil
	}
	m := googleLanguage.FindStringSubmatch(l)
	if m == nil {
		return "", fmt.Errorf("%s %q is not a language code such as en or pt-BR", name, code)
	}
	if m[2] != "" {
		return m[1] + "-" + strings.ToUpper(m[2]), nil
	}
	return m[1], nil
}

// baseLanguage drops the region from a language code: pt-BR becomes pt.
func baseLanguage(code string) string {
	base, _, _ := strings.Cut(code, "-")
	return base
}

// googleURL is the Google search URL for a normalized query.
func googleURL(q SERPQuery) string {
	v := url.Values{"q": {q.Keyword}, "gl": {q.Country}, "hl": {q.Language}, "num": {"10"}, "pws": {"0"}}
	set := func(on bool, key, value string) {
		if on {
			v.Set(key, value)
		}
	}
	set(q.Page > 1, "start", strconv.Itoa((q.Page-1)*10))
	set(q.TimeRange != "", "tbs", "qdr:"+timeRanges[q.TimeRange])
	set(q.SafeSearch, "safe", "active")
	set(q.ResultsLanguage != "", "lr", "lang_"+q.ResultsLanguage)
	set(q.ResultsCountry != "", "cr", "country"+crCountry(q.ResultsCountry))
	set(q.NoAutocorrect, "nfpr", "1")
	set(q.ShowOmitted, "filter", "0")
	set(q.WebOnly, "udm", "14")
	set(q.Near != "", "near", q.Near)
	for k, val := range q.Params {
		v.Set(k, val)
	}
	return "https://" + q.Domain + "/search?" + v.Encode()
}

// crCountry is a country in Google's cr form, which names the United
// Kingdom UK rather than GB.
func crCountry(c string) string {
	if c == "gb" {
		return "UK"
	}
	return strings.ToUpper(c)
}

// googleOnly names the params set on q that the fallback engines ignore.
func (q SERPQuery) googleOnly() []string {
	var names []string
	add := func(on bool, name string) {
		if on {
			names = append(names, name)
		}
	}
	add(q.Domain != "www.google.com", "google_domain")
	add(q.Page > 1, "page")
	add(q.TimeRange != "", "time_range")
	add(q.SafeSearch, "safe_search")
	add(q.ResultsLanguage != "", "results_language")
	add(q.ResultsCountry != "", "results_country")
	add(q.NoAutocorrect, "no_autocorrect")
	add(q.ShowOmitted, "show_omitted")
	add(q.Near != "", "near")
	add(q.WebOnly, "web_only")
	add(len(q.Params) > 0, "params")
	return names
}

// SERP searches Google for the keyword, falling back to DuckDuckGo and then
// Bing when Google shows a bot wall. Engine says which one answered.
func (r *Researcher) SERP(ctx context.Context, q SERPQuery) (SERP, error) {
	q, err := q.normalize()
	if err != nil {
		return SERP{}, err
	}
	kq := q
	kq.Keyword = strings.ToLower(kq.Keyword)
	key := fmt.Sprintf("%s|%v", googleURL(kq), q.ResolveURLs)
	var out SERP
	if r.Cache.Get(serpBucket, key, serpTTL, &out) {
		out.Cached = true
		return out, nil
	}
	var notes []string
	err = r.Browser.Use(ctx, func(p browser.Page) error {
		for _, e := range engines {
			search := e.url(q)
			if _, err := r.navigate(ctx, p, search); err != nil {
				notes = append(notes, e.name+": "+err.Error())
				continue
			}
			var raw rawSERP
			if err := p.Eval(ctx, call(e.js, map[string]string{"q": q.Keyword, "hl": q.Language, "gl": q.Country}), &raw); err != nil {
				notes = append(notes, e.name+": "+err.Error())
				continue
			}
			if raw.Walled || len(raw.Results) < 2 {
				notes = append(notes, e.name+": bot wall or no results")
				continue
			}
			offset := 0
			if e.name == "google" {
				offset = (q.Page - 1) * 10
			} else if ignored := q.googleOnly(); len(ignored) > 0 {
				notes = append(notes, e.name+" ignored "+strings.Join(ignored, ", "))
			}
			out = buildSERP(e.name, q.Keyword, raw, offset)
			out.SearchURL = search
			if q.ResolveURLs {
				r.resolveLinks(ctx, p, out.Results)
			}
			return nil
		}
		return fmt.Errorf("no search engine returned results (%s)", strings.Join(notes, "; "))
	})
	if err != nil {
		return SERP{}, err
	}
	out.Note = strings.Join(notes, "; ")
	r.Cache.Put(serpBucket, key, out)
	return out, nil
}

// resolveLinks follows each redirect link to fill in the exact URL.
func (r *Researcher) resolveLinks(ctx context.Context, p browser.Page, results []Result) {
	for i := range results {
		if results[i].URL != "" {
			continue
		}
		if final, err := r.navigate(ctx, p, results[i].Link); err == nil {
			results[i].URL = final
			if results[i].Domain == "" {
				results[i].Domain = host(final)
			}
		}
	}
}

const maxResults = 10

// buildSERP cleans what the engine page yielded: duplicates removed,
// exact URLs and domains filled in where known, and the keyword itself
// dropped from the suggestion lists. Ranks start after offset.
func buildSERP(engine, keyword string, raw rawSERP, offset int) SERP {
	s := SERP{Keyword: keyword, Engine: engine, Results: []Result{}}
	seen := map[string]bool{}
	for _, rr := range raw.Results {
		if rr.Href == "" || seen[rr.Href] || len(s.Results) == maxResults {
			continue
		}
		seen[rr.Href] = true
		res := Result{Rank: offset + len(s.Results) + 1, Title: rr.Title, Link: rr.Href, Snippet: rr.Snippet, URL: destination(rr)}
		res.Domain, res.Breadcrumb = splitCite(rr.Cite)
		if res.URL != "" && (res.Domain == "" || !strings.Contains(res.Domain, ".")) {
			res.Domain = host(res.URL)
		}
		s.Results = append(s.Results, res)
	}
	s.PeopleAlsoAsk = cleanList(raw.PAA, keyword)
	s.RelatedSearches = cleanList(raw.Related, keyword)
	s.Autocomplete = cleanList(raw.Suggest, keyword)
	s.BoldTerms, s.BoldPassages = boldSpans(raw.Results, keyword)
	return s
}

const (
	maxBoldWords    = 4
	maxBoldTerms    = 30
	maxBoldPassages = 20
)

// boldSpans splits the snippets' bolded spans into terms (up to 4 words,
// no digits), ranked by how many results bold them, and passages: the
// longer answers Google also bolds. The keyword itself and spans of only
// stopwords are dropped. A term keeps the casing it was first seen with.
func boldSpans(results []rawResult, keyword string) ([]BoldTerm, []string) {
	var terms []BoldTerm
	var passages []string
	index := map[string]int{}
	seenPassage := map[string]bool{}
	for _, rr := range results {
		inResult := map[string]bool{}
		for _, span := range rr.Bold {
			span = strings.Join(strings.Fields(span), " ")
			low := strings.ToLower(span)
			if span == "" || low == strings.ToLower(keyword) {
				continue
			}
			words := tokens(span)
			if len(words) > maxBoldWords || strings.ContainsAny(span, "0123456789") {
				if !seenPassage[low] && len(passages) < maxBoldPassages {
					seenPassage[low] = true
					passages = append(passages, span)
				}
				continue
			}
			if !slices.ContainsFunc(words, func(w string) bool { return !stopwords[w] }) || inResult[low] {
				continue
			}
			inResult[low] = true
			if i, ok := index[low]; ok {
				terms[i].Results++
				continue
			}
			index[low] = len(terms)
			terms = append(terms, BoldTerm{Term: span, Results: 1})
		}
	}
	sort.SliceStable(terms, func(i, j int) bool { return terms[i].Results > terms[j].Results })
	if len(terms) > maxBoldTerms {
		terms = terms[:maxBoldTerms]
	}
	return terms, passages
}

// destination returns the result's real URL, or "" when only a redirect
// link is known.
func destination(rr rawResult) string {
	if rr.SURL != "" {
		return rr.SURL
	}
	u, err := url.Parse(rr.Href)
	if err != nil {
		return ""
	}
	h := strings.TrimPrefix(u.Hostname(), "www.")
	switch {
	case h == "bing.com" && strings.HasPrefix(u.Path, "/ck/"):
		enc := strings.TrimPrefix(u.Query().Get("u"), "a1")
		dec, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(enc, "="))
		if err != nil {
			return ""
		}
		return string(dec)
	case strings.HasPrefix(h, "google.") || h == "duckduckgo.com":
		return ""
	}
	return rr.Href
}

// splitCite splits a breadcrumb such as "https://www.allrecipes.com › ...
// › Pan-Fried Chicken" into the domain and the trail after it.
func splitCite(cite string) (domain, trail string) {
	head, rest, _ := strings.Cut(cite, " › ")
	head = strings.TrimSpace(head)
	if !strings.Contains(head, "://") {
		head = "https://" + head
	}
	if h := host(head); strings.Contains(h, ".") && !strings.Contains(h, " ") {
		return h, strings.TrimSpace(rest)
	}
	return "", ""
}

func host(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// cleanList trims, dedupes and drops the keyword itself, page numbers and
// pager links.
func cleanList(in []string, keyword string) []string {
	var out []string
	seen := map[string]bool{strings.ToLower(keyword): true}
	for _, s := range in {
		s = strings.Join(strings.Fields(s), " ")
		low := strings.ToLower(s)
		if s == "" || seen[low] || isNumber(s) || low == "next" || low == "previous" || low == "more results" {
			continue
		}
		seen[low] = true
		out = append(out, s)
	}
	return out
}
