package research

import (
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// FocusWord says where a keyword word, or a word sharing its stem, appears
// in the page's title and first heading.
type FocusWord struct {
	InTitle bool `json:"in_title"`
	InH1    bool `json:"in_h1"`
}

// KeywordLookup is one keyword word or word run looked up as an exact
// Wikipedia title. Result is exact, missing, disambiguation or redirect.
type KeywordLookup struct {
	Phrase string   `json:"phrase"`
	Result string   `json:"result"`
	Title  string   `json:"title,omitempty"`
	Chosen []string `json:"chosen,omitempty"` // options kept by their links to known entities
	Links  int      `json:"links,omitempty"`  // the top link count
}

// titleQID is an article and its Wikidata item.
type titleQID struct{ Title, QID string }

// lookupHit is an entity found for a keyword phrase. How is exact,
// disambiguation or redirect.
type lookupHit struct{ Phrase, Title, QID, How string }

// stopWords are English words that carry no topic. "it", "can" and "do" are
// left out: they are nouns in keywords such as "IT support" and "trash can
// cleaning".
var stopWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`a about an and are as at be by does for from how i in is me my near of on or the to vs what with you your`) {
		stopWords[w] = true
	}
}

// wordsOf splits on anything that isn't a letter or digit and lowercases.
func wordsOf(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// contentWords returns the keyword's words without stop words or repeats,
// in order.
func contentWords(keyword string) []string {
	var out []string
	for _, w := range wordsOf(keyword) {
		if !stopWords[w] && !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	return out
}

var stemSuffixes = []string{"ings", "ing", "ers", "er", "es", "s"}

// stem strips the first of ings, ing, ers, er, es, s that leaves at least
// three letters.
func stem(w string) string {
	for _, suf := range stemSuffixes {
		if s, ok := strings.CutSuffix(w, suf); ok && utf8.RuneCountInString(s) >= 3 {
			return s
		}
	}
	return w
}

// sharesStem reports whether any word of the title has the stem of one of
// the words.
func sharesStem(title string, words []string) bool {
	for _, t := range wordsOf(title) {
		for _, w := range words {
			if stem(t) == stem(w) {
				return true
			}
		}
	}
	return false
}

// keywordGrams returns every run of adjacent words, by start index and then
// length, except runs that start or end with a stop word.
func keywordGrams(keyword string) []string {
	words := wordsOf(keyword)
	var out []string
	for i := range words {
		for j := i; j < len(words); j++ {
			if stopWords[words[i]] || stopWords[words[j]] {
				continue
			}
			if g := strings.Join(words[i:j+1], " "); !slices.Contains(out, g) {
				out = append(out, g)
			}
		}
	}
	return out
}

// focus marks, for each word, whether the title and the H1 hold a word with
// the same stem.
func focus(words []string, title, h1 string) map[string]FocusWord {
	has := func(text string) map[string]bool {
		m := map[string]bool{}
		for _, t := range wordsOf(text) {
			m[stem(t)] = true
		}
		return m
	}
	inTitle, inH1 := has(title), has(h1)
	out := map[string]FocusWord{}
	for _, w := range words {
		out[w] = FocusWord{InTitle: inTitle[stem(w)], InH1: inH1[stem(w)]}
	}
	return out
}

// sameTitle compares Wikipedia titles: underscores are spaces and the first
// letter is case-insensitive, as on Wikipedia; the rest must match exactly.
func sameTitle(a, b string) bool {
	a, b = strings.ReplaceAll(a, "_", " "), strings.ReplaceAll(b, "_", " ")
	ra, na := utf8.DecodeRuneInString(a)
	rb, nb := utf8.DecodeRuneInString(b)
	return strings.EqualFold(string(ra), string(rb)) && a[na:] == b[nb:]
}

func upperFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// option is an article a disambiguation page or redirect may mean.
type option struct{ title, qid string }

// keywordLookups looks every gram up as an exact Wikipedia title; it never
// searches. Exact hits come first. Each disambiguation page and redirect is
// then settled, in gram order, by counting links between its options and
// the known titles plus the exact hits; the options with the most links win,
// when that is at least 1.
func (r *resolver) keywordLookups(grams []string, known []titleQID) ([]KeywordLookup, []lookupHit, error) {
	sent := make([]string, len(grams))
	for i, g := range grams {
		sent[i] = upperFirst(g)
	}
	looks, err := r.titles(sent)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	var knownTitles []string
	for _, k := range known {
		seen[k.QID] = true
		knownTitles = append(knownTitles, k.Title)
	}
	out := make([]KeywordLookup, len(grams))
	options := make([][]option, len(grams))
	var hits []lookupHit
	var urls []string
	var owners []int
	for i, g := range grams {
		l := looks[sent[i]]
		out[i].Phrase = g
		switch {
		case l.page.Missing || l.page.Invalid:
			out[i].Result = "missing"
		case l.page.disambiguation():
			out[i].Result, out[i].Title = "disambiguation", l.page.Title
			urls = append(urls, r.wikiAPI(url.Values{
				"action": {"query"}, "generator": {"links"}, "titles": {l.page.Title}, "gplnamespace": {"0"}, "gpllimit": {"max"},
				"redirects": {"1"}, "prop": {"pageprops|description"}, "ppprop": {"wikibase_item|disambiguation"},
			}))
			owners = append(owners, i)
		case l.redirected:
			out[i].Result, out[i].Title = "redirect", l.page.Title
			options[i] = []option{{l.page.Title, l.page.PageProps["wikibase_item"]}}
		default:
			out[i].Result, out[i].Title = "exact", l.page.Title
			qid := l.page.PageProps["wikibase_item"]
			if qid != "" && !seen[qid] {
				seen[qid] = true
				hits = append(hits, lookupHit{Phrase: g, Title: l.page.Title, QID: qid, How: "exact"})
				knownTitles = append(knownTitles, l.page.Title)
			}
		}
	}
	// Every option on a disambiguation page counts, so follow the links
	// query's continue until it ends.
	for len(urls) > 0 {
		resps := make([]wikiResp, len(urls))
		if err := r.fetch(urls, ptrs(resps)); err != nil {
			return nil, nil, err
		}
		var nextURLs []string
		var nextOwners []int
		for k, resp := range resps {
			for _, pg := range resp.Query.Pages {
				if !pg.Missing && !pg.Invalid && !pg.disambiguation() {
					options[owners[k]] = append(options[owners[k]], option{pg.Title, pg.PageProps["wikibase_item"]})
				}
			}
			if len(resp.Continue) > 0 {
				u, err := url.Parse(urls[k])
				if err != nil {
					return nil, nil, err
				}
				q := u.Query()
				for key, v := range resp.Continue {
					q.Set(key, v)
				}
				u.RawQuery = q.Encode()
				nextURLs, nextOwners = append(nextURLs, u.String()), append(nextOwners, owners[k])
			}
		}
		urls, owners = nextURLs, nextOwners
	}
	var all []string
	for _, opts := range options {
		for _, o := range opts {
			all = append(all, o.title)
		}
	}
	counts, err := r.linkCounts(all, knownTitles)
	if err != nil {
		return nil, nil, err
	}
	for i, opts := range options {
		top := 0
		for _, o := range opts {
			top = max(top, counts[o.title])
		}
		if top < 1 {
			continue
		}
		out[i].Links = top
		var chosen []option
		for _, o := range opts {
			if counts[o.title] == top {
				chosen = append(chosen, o)
			}
		}
		slices.SortFunc(chosen, func(a, b option) int { return strings.Compare(a.title, b.title) })
		for _, o := range chosen {
			out[i].Chosen = append(out[i].Chosen, o.title)
			if o.qid != "" && !seen[o.qid] {
				seen[o.qid] = true
				hits = append(hits, lookupHit{Phrase: out[i].Phrase, Title: o.title, QID: o.qid, How: out[i].Result})
			}
		}
	}
	return out, hits, nil
}

// linkCounts counts, for each option, its links to the known titles plus
// the known titles' links to it.
func (r *resolver) linkCounts(options, known []string) (map[string]int, error) {
	options, known = dedupe(options), dedupe(known)
	counts := map[string]int{}
	isOption := map[string]bool{}
	for _, o := range options {
		counts[o], isOption[o] = 0, true
	}
	type req struct {
		params  url.Values
		reverse bool // the known titles are the sources
	}
	var pending []req
	query := func(from, to []string, reverse bool) {
		pending = append(pending, req{url.Values{
			"action": {"query"}, "prop": {"links"}, "titles": {strings.Join(from, "|")},
			"pltitles": {strings.Join(to, "|")}, "pllimit": {"max"}, "redirects": {"1"},
		}, reverse})
	}
	for i := 0; i < len(options); i += batchSize {
		ob := options[i:min(i+batchSize, len(options))]
		for j := 0; j < len(known); j += batchSize {
			kb := known[j:min(j+batchSize, len(known))]
			query(ob, kb, false)
			query(kb, ob, true)
		}
	}
	for len(pending) > 0 {
		urls := make([]string, len(pending))
		for i, q := range pending {
			urls[i] = r.wikiAPI(cloneValues(q.params))
		}
		resps := make([]wikiResp, len(urls))
		if err := r.fetch(urls, ptrs(resps)); err != nil {
			return nil, err
		}
		var next []req
		for i, resp := range resps {
			for _, pg := range resp.Query.Pages {
				for _, l := range pg.Links {
					switch {
					case pending[i].reverse && isOption[l.Title]:
						counts[l.Title]++
					case !pending[i].reverse && isOption[pg.Title]:
						counts[pg.Title]++
					}
				}
			}
			if len(resp.Continue) > 0 {
				p := cloneValues(pending[i].params)
				for k, v := range resp.Continue {
					p.Set(k, v)
				}
				next = append(next, req{p, pending[i].reverse})
			}
		}
		pending = next
	}
	return counts, nil
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vs := range v {
		out[k] = slices.Clone(vs)
	}
	return out
}
