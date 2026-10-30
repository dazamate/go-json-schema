package research

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/dazamate/go-json-schema/internal/textrazor"
)

// EntityLinker finds the Wikipedia-linked entities and topics in a text.
// textrazor.Client is one. It is the one research call made outside the
// browser: it gets only text the browser already fetched.
type EntityLinker interface {
	Analyze(ctx context.Context, text string) (textrazor.Result, error)
}

// PageEntity is an entity the ranking pages mention, linked in context by
// TextRazor, so its name is the exact Wikipedia title it means.
type PageEntity struct {
	Name        string   `json:"name"`                   // Wikipedia title in the pages' language; pass it to resolve_entities
	EnglishName string   `json:"english_name,omitempty"` // English Wikipedia title, when different
	Wikipedia   string   `json:"wikipedia,omitempty"`
	QID         string   `json:"qid,omitempty"`
	Matched     []string `json:"matched,omitempty"` // words the pages used for it
	Mentions    int      `json:"mentions"`
	Pages       int      `json:"pages"`
	Relevance   float64  `json:"relevance"`  // highest TextRazor relevance, 0 to 1
	Confidence  float64  `json:"confidence"` // highest TextRazor confidence, about 0.5 to 10
	Score       float64  `json:"score"`
	Types       []string `json:"types,omitempty"` // DBpedia types
	Rho         *float64 `json:"rho,omitempty"`   // see Correlate
	Significant bool     `json:"significant,omitempty"`
}

// PageTopic is a theme TextRazor found across the ranking pages.
type PageTopic struct {
	Label     string  `json:"label"`
	Wikipedia string  `json:"wikipedia,omitempty"`
	QID       string  `json:"qid,omitempty"`
	Pages     int     `json:"pages"`
	Score     float64 `json:"score"` // mean over the pages, 0 to 1
}

const (
	// DefaultLinkMaxBytes is the text sent per page: 3 TextRazor requests.
	DefaultLinkMaxBytes = 3 * textrazor.UnitBytes
	maxEntities         = 60
	maxTopics           = 30
	maxMatched          = 10
	linkWorkers         = 2 // TextRazor's free plan allows 2 concurrent requests
	noLinkerNote        = "no entities: set textrazor.api_key in the schema-mcp config file to link the pages' entities with TextRazor"
)

// linkEntities sends each page's title and text to the linker, two at a
// time, and merges what it finds across pages. A failure on some pages is
// reported in the note and the rest are still merged.
func (r *Researcher) linkEntities(ctx context.Context, docs []Doc, minPages int) ([]PageEntity, []PageTopic, string) {
	if r.Linker == nil {
		return nil, nil, noLinkerNote
	}
	if len(docs) == 0 {
		return nil, nil, ""
	}
	max := r.LinkMaxBytes
	if max <= 0 {
		max = DefaultLinkMaxBytes
	}
	results := make([]*textrazor.Result, len(docs))
	errs := make([]error, len(docs))
	units, cached := 0, 0
	progress(ctx, fmt.Sprintf("linking the entities on %d pages with TextRazor", len(docs)))
	var mu sync.Mutex
	sem := make(chan struct{}, linkWorkers)
	var wg sync.WaitGroup
	for i, d := range docs {
		text := textrazor.Clip(strings.TrimSpace(d.Title+"\n\n"+d.Text), max)
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res, hit, err := r.analyze(ctx, text)
			if err != nil {
				errs[i] = err
				return
			}
			mu.Lock()
			if hit {
				cached++
			} else {
				units += res.Units
			}
			mu.Unlock()
			results[i] = &res
		}()
	}
	wg.Wait()

	var notes []string
	failed := 0
	var firstErr error
	for _, err := range errs {
		if err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if units > 0 || cached > 0 {
		notes = append(notes, fmt.Sprintf("linked %d pages with %d TextRazor requests (one per 10 KB of text sent), %d pages cached", len(docs)-failed-cached, units, cached))
	}
	if failed > 0 {
		notes = append(notes, fmt.Sprintf("TextRazor failed on %d of %d pages: %v", failed, len(docs), firstErr))
	}
	ents, topics := mergeLinked(results, minPages)
	return ents, topics, strings.Join(notes, "; ")
}

// textrazorKey is the cache key of text's TextRazor answer.
func textrazorKey(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// analyze sends text to the linker, or returns the cached result for the
// same text. A failed call is not cached.
func (r *Researcher) analyze(ctx context.Context, text string) (res textrazor.Result, cached bool, err error) {
	key := textrazorKey(text)
	if r.Cache.Get("textrazor", key, pageTTL, &res) {
		return res, true, nil
	}
	if r.Linker == nil {
		return textrazor.Result{}, false, errors.New("no entity linker configured")
	}
	res, err = r.Linker.Analyze(ctx, text)
	if err != nil {
		return textrazor.Result{}, false, err
	}
	r.Cache.Put("textrazor", key, res)
	return res, false, nil
}

// mergeLinked merges the per-page results, given in rank order, into one
// entity list and one topic list. A nil result is a page TextRazor failed
// on: it is left out of the rank correlations and topic means, not counted
// as a page without mentions. An entity is keyed by its Wikidata item,
// else its title; mentions TextRazor could not link (numbers, dates) are
// dropped. An entity's score sums, over the pages naming it, a damped
// mention count (1+ln mentions) times (1+relevance), weighted down by rank
// like RankTerms.
func mergeLinked(results []*textrazor.Result, minPages int) ([]PageEntity, []PageTopic) {
	type entStat struct {
		e       PageEntity
		perPage []float64
		seen    map[string]bool
	}
	type topicStat struct {
		t   PageTopic
		sum float64
	}
	// slot[i] is page i's place among the linked pages.
	slot := make([]int, len(results))
	linked := 0
	for i, res := range results {
		if res != nil {
			slot[i] = linked
			linked++
		}
	}
	ents := map[string]*entStat{}
	topics := map[string]*topicStat{}
	for i, res := range results {
		if res == nil {
			continue
		}
		weight := 1 / (1 + 0.15*float64(i))
		count := map[string]int{}
		rel := map[string]float64{}
		for _, m := range res.Entities {
			if m.ID == "" && m.WikiLink == "" {
				continue
			}
			key := m.WikidataID
			if key == "" {
				key = strings.ToLower(m.ID)
			}
			s := ents[key]
			if s == nil {
				s = &entStat{e: PageEntity{Name: m.ID, QID: m.WikidataID, Wikipedia: articleLink(m.WikiLink), Types: m.Types},
					perPage: make([]float64, linked), seen: map[string]bool{}}
				if m.EnglishID != "" && m.EnglishID != m.ID {
					s.e.EnglishName = m.EnglishID
				}
				ents[key] = s
			}
			if w := strings.Join(strings.Fields(m.MatchedText), " "); w != "" && !s.seen[strings.ToLower(w)] && len(s.e.Matched) < maxMatched {
				s.seen[strings.ToLower(w)] = true
				s.e.Matched = append(s.e.Matched, w)
			}
			s.e.Mentions++
			s.e.Relevance = math.Max(s.e.Relevance, m.Relevance)
			s.e.Confidence = math.Max(s.e.Confidence, m.Confidence)
			count[key]++
			rel[key] = math.Max(rel[key], m.Relevance)
		}
		for key, n := range count {
			s := ents[key]
			s.e.Pages++
			s.perPage[slot[i]] = float64(n)
			s.e.Score += weight * (1 + math.Log(float64(n))) * (1 + rel[key])
		}
		onPage := map[string]bool{}
		for _, t := range res.Topics {
			key := strings.ToLower(t.Label)
			if key == "" || onPage[key] {
				continue
			}
			onPage[key] = true
			s := topics[key]
			if s == nil {
				s = &topicStat{t: PageTopic{Label: t.Label}}
				if link := articleLink(t.WikiLink); link != "" {
					s.t.Wikipedia, s.t.QID = link, t.WikidataID
				}
				topics[key] = s
			}
			s.t.Pages++
			s.sum += t.Score
		}
	}
	var outE []PageEntity
	for _, s := range ents {
		if s.e.Pages < minPages {
			continue
		}
		s.e.Score = math.Round(s.e.Score*100) / 100
		s.e.Rho, s.e.Significant = Correlate(s.perPage)
		outE = append(outE, s.e)
	}
	sort.Slice(outE, func(i, j int) bool {
		if outE[i].Score != outE[j].Score {
			return outE[i].Score > outE[j].Score
		}
		return outE[i].Name < outE[j].Name
	})
	var outT []PageTopic
	for _, s := range topics {
		if s.t.Pages < minPages {
			continue
		}
		s.t.Score = math.Round(s.sum/float64(linked)*100) / 100
		outT = append(outT, s.t)
	}
	sort.Slice(outT, func(i, j int) bool {
		if outT[i].Score != outT[j].Score {
			return outT[i].Score > outT[j].Score
		}
		return outT[i].Label < outT[j].Label
	})
	return outE[:min(len(outE), maxEntities)], outT[:min(len(outT), maxTopics)]
}

// articleLink returns a Wikipedia article link over https, or "" for a
// category page or anything that is not a Wikipedia link. TextRazor's topic
// links leave out /wiki/ (http://en.wikipedia.org/Bride), which Wikipedia
// answers with "page not found", so it is put back.
func articleLink(link string) string {
	u, err := url.Parse(link)
	if err != nil || !strings.HasSuffix(u.Hostname(), "wikipedia.org") || strings.Contains(u.Path, "Category:") {
		return ""
	}
	if !strings.HasPrefix(u.Path, "/wiki/") {
		title := strings.TrimPrefix(u.Path, "/")
		if title == "" {
			return ""
		}
		// keep the title as written: (honor), not %28honor%29
		u.Path, u.RawPath = "/wiki/"+title, "/wiki/"+strings.TrimPrefix(u.EscapedPath(), "/")
	}
	u.Scheme = "https"
	return u.String()
}
