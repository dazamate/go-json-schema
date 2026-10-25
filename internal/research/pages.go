package research

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/dazamate/go-json-schema/internal/browser"
)

// PageQuery is a set of ranking pages to read, in search-rank order.
type PageQuery struct {
	Links    []string // result URLs or Google redirect links
	MaxTerms int      // per term list; default 40
	MinPages int      // see TermOptions
}

// PageReport is what the ranking pages say and which schema they publish.
type PageReport struct {
	Pages       []Page       `json:"pages"`
	Words       []Term       `json:"words"`
	Phrases     []Term       `json:"phrases"`
	SchemaTerms []SchemaTerm `json:"schema_terms,omitempty"`
	Entities    []PageEntity `json:"entities,omitempty"`
	Topics      []PageTopic  `json:"topics,omitempty"`
	EntityNote  string       `json:"entity_note,omitempty"`
}

// Page is one fetched ranking page.
type Page struct {
	Link        string      `json:"link"`
	URL         string      `json:"url,omitempty"`
	Canonical   string      `json:"canonical,omitempty"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Lang        string      `json:"lang,omitempty"`
	Headings    []string    `json:"headings,omitempty"`
	WordCount   int         `json:"word_count,omitempty"`
	Schema      *PageSchema `json:"schema,omitempty"`
	Cached      bool        `json:"cached,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// PageSchema summarizes a page's JSON-LD.
type PageSchema struct {
	Types      []string `json:"types"`
	About      []string `json:"about,omitempty"`
	Mentions   []string `json:"mentions,omitempty"`
	KnowsAbout []string `json:"knows_about,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
}

// SchemaTerm is an entity or keyword competitors put in their schema.
type SchemaTerm struct {
	Name  string   `json:"name"`
	Pages int      `json:"pages"`
	As    []string `json:"as"` // about, mentions, knowsAbout, keywords
}

const maxLinks = 20

type rawPage struct {
	Href        string   `json:"href"`
	Canonical   string   `json:"canonical"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Lang        string   `json:"lang"`
	Headings    []string `json:"headings"`
	LD          []string `json:"ld"`
	Text        string   `json:"text"`
}

type cachedPage struct {
	Page Page   `json:"page"`
	Text string `json:"text"`
}

// pageJS reads the page's facts, then its main text with navigation,
// footers, sidebars and comments hidden. Hidden chrome is skipped when it
// holds most of the text, as with pages wrapped in one big form.
const pageJS = `(() => {
  const meta = n => { const m = document.querySelector('meta[name="' + n + '"], meta[property="' + n + '"]'); return m ? (m.content || '') : ''; };
  const chrome = 'nav, footer, aside, [role=navigation], [role=contentinfo], [role=complementary], #comments, .comments, .comment-list, script, style, noscript, iframe, form, template';
  const headings = [...document.querySelectorAll('h1, h2, h3')].filter(h => !h.closest(chrome))
    .map(h => h.tagName.toLowerCase() + ': ' + (h.innerText || '').trim().replace(/\s+/g, ' ').slice(0, 120)).filter(s => s.length > 4).slice(0, 40);
  const ld = [...document.querySelectorAll('script[type="application/ld+json"]')].map(s => s.textContent.slice(0, 300000));
  let root = document.body;
  for (const el of document.querySelectorAll('main, [role=main], article')) {
    const n = (el.innerText || '').length;
    if (n >= 500 && (root === document.body || n > (root.innerText || '').length)) root = el;
  }
  const total = (root.innerText || '').length;
  const hide = el => { if (!el.contains(root) && (el.innerText || '').length < total / 2) el.style.setProperty('display', 'none', 'important'); };
  document.querySelectorAll(chrome).forEach(hide);
  document.querySelectorAll('header').forEach(el => { if (!root.contains(el)) hide(el); });
  return {href: location.href, canonical: (document.querySelector('link[rel=canonical]') || {}).href || '',
    title: document.title, description: meta('description') || meta('og:description'), lang: document.documentElement.lang || '',
    headings, ld, text: (root.innerText || '').slice(0, 150000)};
})()`

// PageTerms reads each page and ranks the words and phrases they share,
// and, with a Linker, the entities and topics they mention.
func (r *Researcher) PageTerms(ctx context.Context, q PageQuery) (PageReport, error) {
	if len(q.Links) == 0 {
		return PageReport{}, errors.New("no links given")
	}
	if len(q.Links) > maxLinks {
		return PageReport{}, fmt.Errorf("%d links given; at most %d per call", len(q.Links), maxLinks)
	}
	cached := make([]*cachedPage, len(q.Links))
	var todo []int
	for i, link := range q.Links {
		var c cachedPage
		if r.Cache.Get("page", link, pageTTL, &c) {
			c.Page.Link, c.Page.Cached = link, true
			cached[i] = &c
		} else {
			todo = append(todo, i)
		}
	}
	// Google links wait for the cooldown, so read every other site first.
	slices.SortStableFunc(todo, func(a, b int) int {
		return cmp.Compare(b2i(isGoogle(q.Links[a])), b2i(isGoogle(q.Links[b])))
	})
	if len(todo) > 0 {
		err := r.Browser.Use(ctx, func(p browser.Page) error {
			prev := ""
			for n, i := range todo {
				progress(ctx, fmt.Sprintf("reading page %d of %d: %s", n+1, len(todo), q.Links[i]))
				c := r.fetchPage(ctx, p, q.Links[i], prev)
				cached[i] = &c
				if c.Page.Error == "" {
					prev = c.Page.URL
					r.Cache.Put("page", q.Links[i], c)
					if c.Page.URL != q.Links[i] {
						r.Cache.Put("page", c.Page.URL, c)
					}
				}
			}
			return nil
		})
		if err != nil {
			return PageReport{}, err
		}
	}
	rep := PageReport{Pages: make([]Page, len(q.Links))}
	var docs []Doc
	var schemas []*PageSchema
	for i, c := range cached {
		rep.Pages[i] = c.Page
		if c.Page.Error != "" {
			continue
		}
		var heads []string
		for _, h := range c.Page.Headings {
			_, text, _ := strings.Cut(h, ": ")
			heads = append(heads, text)
		}
		docs = append(docs, Doc{Title: c.Page.Title, Headings: heads, Text: c.Text})
		schemas = append(schemas, c.Page.Schema)
	}
	rep.Words, rep.Phrases = RankTerms(docs, TermOptions{Max: q.MaxTerms, MinPages: q.MinPages})
	rep.SchemaTerms = mergeSchemaTerms(schemas)
	rep.Entities, rep.Topics, rep.EntityNote = r.linkEntities(ctx, docs, minPages(q.MinPages, len(docs)))
	return rep, nil
}

// fetchPage loads one page. prev is the URL the tab showed before, so a
// navigation that failed without leaving it is not read as this page.
func (r *Researcher) fetchPage(ctx context.Context, p browser.Page, link, prev string) cachedPage {
	c := cachedPage{Page: Page{Link: link}}
	if u, err := url.Parse(link); err == nil && strings.HasSuffix(strings.ToLower(u.Path), ".pdf") {
		c.Page.Error = "skipped: PDF"
		return c
	}
	_, navErr := r.navigate(ctx, p, link)
	var te *browser.ToolError
	if navErr != nil && !errors.As(navErr, &te) {
		c.Page.Error = navErr.Error()
		return c
	}
	// A page that never fired its load event may still be readable.
	var raw rawPage
	if err := p.Eval(ctx, pageJS, &raw); err != nil {
		c.Page.Error = err.Error()
		return c
	}
	if navErr != nil && (raw.Href == prev || raw.Href == "about:blank") {
		c.Page.Error = navErr.Error()
		return c
	}
	c.Page.URL = raw.Href
	c.Page.Canonical = raw.Canonical
	c.Page.Title = strings.TrimSpace(raw.Title)
	c.Page.Description = strings.TrimSpace(raw.Description)
	c.Page.Lang = raw.Lang
	c.Page.Headings = raw.Headings
	c.Page.WordCount = len(strings.Fields(raw.Text))
	if len(raw.LD) > 0 {
		c.Page.Schema = summarizeJSONLD(raw.LD)
	}
	c.Text = raw.Text
	return c
}

const maxSchemaList = 50

// machineTag matches CMS tags such as "contentId: 55a8..." that some
// publishers put in keywords.
var machineTag = regexp.MustCompile(`^[A-Za-z][\w-]*: `)

// summarizeJSONLD lists the types and the about, mentions, knowsAbout and
// keywords entries in a page's JSON-LD blocks. Blocks that are not JSON
// are skipped.
func summarizeJSONLD(blocks []string) *PageSchema {
	s := &PageSchema{Types: []string{}}
	lists := map[string]*[]string{"about": &s.About, "mentions": &s.Mentions, "knowsAbout": &s.KnowsAbout, "keywords": &s.Keywords}
	seen := map[string]map[string]bool{}
	add := func(list, v string) {
		v = strings.Join(strings.Fields(v), " ")
		if v == "" || len(*lists[list]) >= maxSchemaList || (list == "keywords" && machineTag.MatchString(v)) {
			return
		}
		if seen[list] == nil {
			seen[list] = map[string]bool{}
		}
		if low := strings.ToLower(v); !seen[list][low] {
			seen[list][low] = true
			*lists[list] = append(*lists[list], v)
		}
	}
	seenType := map[string]bool{}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, t := range strs(x["@type"]) {
				if !seenType[t] {
					seenType[t] = true
					s.Types = append(s.Types, t)
				}
			}
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if _, ok := lists[k]; ok {
					for _, name := range entryNames(k, x[k]) {
						add(k, name)
					}
				}
				walk(x[k])
			}
		}
	}
	for _, b := range blocks {
		var v any
		if json.Unmarshal([]byte(b), &v) == nil {
			walk(v)
		}
	}
	return s
}

// entryNames returns the names in an about/mentions/knowsAbout/keywords
// value: plain strings (keywords split at commas), or nodes by name, else
// by @id.
func entryNames(key string, v any) []string {
	var out []string
	var items []any
	if arr, ok := v.([]any); ok {
		items = arr
	} else {
		items = []any{v}
	}
	for _, it := range items {
		switch x := it.(type) {
		case string:
			if key == "keywords" {
				out = append(out, strings.Split(x, ",")...)
			} else {
				out = append(out, x)
			}
		case map[string]any:
			if n := strs(x["name"]); len(n) > 0 {
				out = append(out, n[0])
			} else if id, ok := x["@id"].(string); ok {
				out = append(out, id)
			}
		}
	}
	return out
}

func strs(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

const maxSchemaTerms = 60

// mergeSchemaTerms counts, across pages, how many use each schema entry,
// matching names ignoring case.
func mergeSchemaTerms(schemas []*PageSchema) []SchemaTerm {
	byName := map[string]*SchemaTerm{}
	var order []string
	for _, s := range schemas {
		if s == nil {
			continue
		}
		onPage := map[string]bool{}
		for _, role := range []struct {
			as    string
			names []string
		}{{"about", s.About}, {"mentions", s.Mentions}, {"knowsAbout", s.KnowsAbout}, {"keywords", s.Keywords}} {
			for _, n := range role.names {
				low := strings.ToLower(n)
				t := byName[low]
				if t == nil {
					t = &SchemaTerm{Name: n}
					byName[low] = t
					order = append(order, low)
				}
				if !onPage[low] {
					onPage[low] = true
					t.Pages++
				}
				if !slices.Contains(t.As, role.as) {
					t.As = append(t.As, role.as)
				}
			}
		}
	}
	out := make([]SchemaTerm, 0, len(order))
	for _, low := range order {
		out = append(out, *byName[low])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Pages > out[j].Pages })
	if len(out) > maxSchemaTerms {
		out = out[:maxSchemaTerms]
	}
	return out
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
