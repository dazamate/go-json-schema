package research

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Doc is one ranking page's text, in search-rank order.
type Doc struct {
	Title    string
	Headings []string
	Text     string
}

// Term is a word or phrase found on the ranking pages.
type Term struct {
	Term        string   `json:"term"`
	Count       int      `json:"count"`                 // uses across all pages
	Pages       int      `json:"pages"`                 // pages using it
	Score       float64  `json:"score"`                 // relevance; see RankTerms
	Rho         *float64 `json:"rho,omitempty"`         // see Correlate
	Significant bool     `json:"significant,omitempty"` // |rho| passes the 95% test
}

// minCorrelationPages is the fewest pages a rank correlation is taken on.
const minCorrelationPages = 5

// TermOptions limits RankTerms' output.
type TermOptions struct {
	Max      int // terms per list; default 40
	MinPages int // pages a term must appear on; default 2 with 3+ pages, else 1
}

// RankTerms extracts single words and 2-3 word phrases from the docs, like
// Zizta's frequency and relevance lists. A term's score sums, over the
// pages using it, a damped use count (1+ln uses), weighted down by search
// rank and up when the term is in the page's title or headings. Phrases
// never cross sentence or comma boundaries and never start or end on a
// stopword. A 2-word phrase is folded into a 3-word phrase that accounts
// for most of its uses. With 5 or more pages, each term also gets its rank
// correlation (see Correlate).
func RankTerms(docs []Doc, opts TermOptions) (words, phrases []Term) {
	if opts.Max <= 0 {
		opts.Max = 40
	}
	opts.MinPages = minPages(opts.MinPages, len(docs))
	type stat struct {
		count, pages int
		score        float64
		uses         []float64 // per page, in rank order
	}
	stats := map[string]*stat{}
	for i, d := range docs {
		weight := 1 / (1 + 0.15*float64(i))
		uses := countTerms(d.Text)
		title := presentTerms(d.Title)
		heads := presentTerms(strings.Join(d.Headings, "\n"))
		for t := range title {
			uses[t] = max(uses[t], 1)
		}
		for t := range heads {
			uses[t] = max(uses[t], 1)
		}
		for t, n := range uses {
			s := stats[t]
			if s == nil {
				s = &stat{uses: make([]float64, len(docs))}
				stats[t] = s
			}
			s.uses[i] = float64(n)
			s.count += n
			s.pages++
			boost := 1.0
			if title[t] {
				boost++
			}
			if heads[t] {
				boost++
			}
			s.score += weight * (1 + math.Log(float64(n))) * boost
		}
	}
	// longest[b] is the highest use count of a 3-word phrase starting or
	// ending with the 2-word phrase b.
	longest := map[string]int{}
	for t, s := range stats {
		if words := strings.Fields(t); len(words) == 3 {
			for _, b := range []string{words[0] + " " + words[1], words[1] + " " + words[2]} {
				longest[b] = max(longest[b], s.count)
			}
		}
	}
	for t, s := range stats {
		if s.pages < opts.MinPages {
			continue
		}
		term := Term{Term: t, Count: s.count, Pages: s.pages, Score: math.Round(s.score*100) / 100}
		term.Rho, term.Significant = Correlate(s.uses)
		switch n := strings.Count(t, " ") + 1; {
		case n == 1:
			words = append(words, term)
		case n == 2 && 4*longest[t] >= 3*s.count:
			// Folded into a 3-word phrase holding at least 3/4 of its uses.
		default:
			phrases = append(phrases, term)
		}
	}
	return top(words, opts.Max), top(phrases, opts.Max)
}

// Correlate returns the Spearman rank correlation between a term's use on
// each page, in search-rank order, and the pages' rank positions, the way
// Cora SEO Software correlates its factors. As in Cora, a negative rho
// means higher-ranked pages use the term more. Significant uses the normal
// approximation of the two-tailed 95% critical value, 1.96/sqrt(n-1). rho
// is nil with fewer than 5 pages or when use is the same on every page.
func Correlate(uses []float64) (rho *float64, significant bool) {
	if len(uses) < minCorrelationPages {
		return nil, false
	}
	r, ok := spearman(uses)
	if !ok {
		return nil, false
	}
	r = math.Round(r*1000) / 1000
	return &r, math.Abs(r) >= 1.96/math.Sqrt(float64(len(uses)-1))
}

// spearman correlates ys with their positions 1..n: the Pearson
// correlation of the ranks, with tied values given their average rank.
// ok is false when ys are all equal.
func spearman(ys []float64) (rho float64, ok bool) {
	n := len(ys)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return ys[order[a]] < ys[order[b]] })
	ranks := make([]float64, n)
	for i := 0; i < n; {
		j := i
		for j+1 < n && ys[order[j+1]] == ys[order[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			ranks[order[k]] = avg
		}
		i = j + 1
	}
	mean := float64(n+1) / 2
	var sxy, sxx, syy float64
	for i, ry := range ranks {
		dx, dy := float64(i+1)-mean, ry-mean
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if syy == 0 {
		return 0, false
	}
	return sxy / math.Sqrt(sxx*syy), true
}

// minPages is the pages a term must appear on: n when set, else 2 with 3
// or more pages and 1 below that.
func minPages(n, docs int) int {
	switch {
	case n > 0:
		return n
	case docs >= 3:
		return 2
	}
	return 1
}

func top(ts []Term, n int) []Term {
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].Score != ts[j].Score {
			return ts[i].Score > ts[j].Score
		}
		if ts[i].Count != ts[j].Count {
			return ts[i].Count > ts[j].Count
		}
		return ts[i].Term < ts[j].Term
	})
	if len(ts) > n {
		ts = ts[:n]
	}
	return ts
}

func presentTerms(text string) map[string]bool {
	out := map[string]bool{}
	for t := range countTerms(text) {
		out[t] = true
	}
	return out
}

// countTerms counts the words and 2-3 word phrases in text.
func countTerms(text string) map[string]int {
	out := map[string]int{}
	for _, seg := range segments(text) {
		toks := tokens(seg)
		for i, tok := range toks {
			if isWord(tok) {
				out[tok]++
			}
			for n := 2; n <= 3 && i+n <= len(toks); n++ {
				gram := toks[i : i+n]
				if isEdge(gram[0]) && isEdge(gram[n-1]) && (n == 2 || !isNumber(gram[1])) {
					out[strings.Join(gram, " ")]++
				}
			}
		}
	}
	return out
}

// segments splits text at line breaks and punctuation that ends a phrase.
func segments(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '\n', '\r', '.', '!', '?', ';', ':', ',', '(', ')', '[', ']', '{', '}', '|', '"', '“', '”', '•', '·', '/', '–', '—':
			return true
		}
		return false
	})
}

// tokens lowercases seg and splits it into words, keeping inner
// apostrophes and hyphens (grandma's, stir-fry).
func tokens(seg string) []string {
	seg = strings.ToLower(strings.ReplaceAll(seg, "’", "'"))
	var out []string
	var b strings.Builder
	flush := func() {
		if w := strings.Trim(b.String(), "'-"); w != "" {
			out = append(out, w)
		}
		b.Reset()
	}
	for _, r := range seg {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case (r == '\'' || r == '-') && b.Len() > 0:
			b.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return out
}

func isWord(tok string) bool {
	return len([]rune(tok)) >= 3 && !isNumber(tok) && !stopwords[tok]
}

func isEdge(tok string) bool {
	return len([]rune(tok)) >= 2 && !isNumber(tok) && !stopwords[tok]
}

func isNumber(tok string) bool {
	for _, r := range tok {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// stopwords are English function words plus page chrome words that are
// never a topic.
var stopwords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`
a about above after again against all almost also am among an and any are aren't around as at
be became because been before being below between both but by can can't cannot could couldn't
did didn't do does doesn't doing don't down during each either else enough etc even ever every
few for from further get gets getting got had hadn't has hasn't have haven't having he he'd he'll
he's her here here's hers herself him himself his how how's however i i'd i'll i'm i've if in into
is isn't it it's its itself just least less let let's like made make makes many may me might more
most much must mustn't my myself near need needs never no nor not now of off often on once one only
or other others our ours ourselves out over own per perhaps put quite rather really same says see
seen shall shan't she she'd she'll she's should shouldn't since so some such sure than that that's
the their theirs them themselves then there there's these they they'd they'll they're they've this
those though through thus to too under until up upon us use used uses using very via was wasn't way
we we'd we'll we're we've well were weren't what what's when when's where where's whether which
while who who's whom whose why why's will with within without won't would wouldn't yet you you'd
you'll you're you've your yours yourself yourselves
cookie cookies newsletter subscribe login logout signup menu skip jump advertisement copyright
`) {
		m[w] = true
	}
	return m
}()
