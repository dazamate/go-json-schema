package research

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dazamate/go-json-schema/internal/azurelang"
	"github.com/dazamate/go-json-schema/internal/browser"
	"github.com/dazamate/go-json-schema/internal/textrazor"
	"github.com/dazamate/go-json-schema/schema"
)

// Where a selected or dropped entity came from.
const (
	SourcePage    = "page"    // linked on our page by TextRazor
	SourceKeyword = "keyword" // a keyword word or phrase looked up as a Wikipedia title
	SourceTheme   = "theme"   // a TextRazor topic of the page
)

// Reasons an entity is dropped, besides the identity reasons.
const (
	ReasonBelowRelevance  = "below_relevance"
	ReasonBelowTopic      = "below_topic"
	ReasonBelowSimilarity = "below_similarity"
	ReasonNotInPageText   = "not_in_page_text" // a mentions entity named only in the title
)

// SelectSettings are the cutoffs select_entities applies, each 0 to 1.
type SelectSettings struct {
	MinRelevance         float64 `json:"min_relevance"`          // lowest TextRazor relevance for a page entity
	MinTopic             float64 `json:"min_topic"`              // lowest topic score for a page entity
	MinKeywordSimilarity float64 `json:"min_keyword_similarity"` // lowest theme-to-keyword cosine
	MinPageSimilarity    float64 `json:"min_page_similarity"`    // lowest theme-to-page cosine
}

// DefaultSelectSettings are our judgement, set by eye from one calibration
// page, not from any Google source.
var DefaultSelectSettings = SelectSettings{0.3, 0.5, 0.40, 0.55}

// SelectQuery is our page and its target keyword.
type SelectQuery struct {
	URL, Keyword string
	Language     string // Wikipedia edition; default the page's lang, else en
	Settings     SelectSettings
}

// SelectReport is the entities chosen for our page's about and mentions,
// with what was looked up and dropped on the way.
type SelectReport struct {
	URL      string `json:"url"`
	FinalURL string `json:"final_url"`
	Title    string `json:"title"`
	Language string `json:"language"`
	Words    int    `json:"words"`

	Keyword string               `json:"keyword"`
	Focus   map[string]FocusWord `json:"focus"`

	About          []Selected      `json:"about"`
	Mentions       []Selected      `json:"mentions"`
	KeywordLookups []KeywordLookup `json:"keyword_lookups"`
	Dropped        []Dropped       `json:"dropped"`

	Settings  SelectSettings `json:"settings"`
	TextRazor TextRazorUsage `json:"textrazor"`
	Azure     *AzureUsage    `json:"azure,omitempty"`
	Model     string         `json:"model"` // the embedding model that scored the themes
	Note      string         `json:"note"`
}

// TextRazorUsage is what this call cost at TextRazor.
type TextRazorUsage struct {
	Units  int  `json:"units"` // requests billed; 0 when cached
	Cached bool `json:"cached"`
}

// AzureUsage is what this call cost at Azure AI Language.
type AzureUsage struct {
	Records int  `json:"records"` // text records billed; 0 when cached
	Cached  bool `json:"cached"`
}

// Dropped is an entity left out, with the measurements that left it out.
type Dropped struct {
	Name              string   `json:"name"`
	QID               string   `json:"qid"`
	Source            string   `json:"source"`
	Relevance         *float64 `json:"relevance,omitempty"`
	Topic             *float64 `json:"topic,omitempty"`
	KeywordSimilarity *float64 `json:"keyword_similarity,omitempty"`
	PageSimilarity    *float64 `json:"page_similarity,omitempty"`
	Reason            string   `json:"reason"`
}

// Selected is one entity chosen for about or mentions: its identity, the
// scores that qualified it and its ready schema.org node.
type Selected struct {
	Name      string `json:"name"`
	QID       string `json:"qid"`
	Source    string `json:"source"`
	Wikipedia string `json:"wikipedia"`

	KGMID        string `json:"kgmid,omitempty"`
	GoogleKGID   string `json:"google_kg_id,omitempty"`
	KGMIDChecked bool   `json:"kgmid_checked"`
	BingID       string `json:"bing_id,omitempty"`
	BingChecked  bool   `json:"bing_checked"`

	Relevance         *float64 `json:"relevance,omitempty"`
	Topic             *float64 `json:"topic,omitempty"`
	Confidence        *float64 `json:"confidence,omitempty"`
	KeywordSimilarity *float64 `json:"keyword_similarity,omitempty"`
	PageSimilarity    *float64 `json:"page_similarity,omitempty"`
	Mentions          int      `json:"mentions,omitempty"`

	Matched     []string `json:"matched"`
	LinkedTo    []string `json:"linked_to"`
	QualifiedBy []string `json:"qualified_by"`

	Thing json.RawMessage `json:"thing"`
}

// SelectEntities chooses our page's about and mentions entities. It reads
// the page through the browser, links it with TextRazor and keeps the page
// entities that pass the score gate or have an own-site link. about is the
// keyword's entities (page entities sharing its stems, then its words looked
// up as exact Wikipedia titles), then the TextRazor topics close to both the
// keyword and the page; mentions is every other kept page entity. Every
// entity is checked by its Wikidata item, and Azure adds Bing IDs when set.
func (r *Researcher) SelectEntities(ctx context.Context, q SelectQuery) (SelectReport, error) {
	if err := q.check(); err != nil {
		return SelectReport{}, err
	}
	if r.Linker == nil {
		return SelectReport{}, errors.New(noLinkerNote)
	}
	// Our page is never served from the page cache, so an edited page is
	// always read fresh.
	var page cachedPage
	err := r.Browser.Use(ctx, func(p browser.Page) error {
		progress(ctx, "reading "+q.URL)
		page = r.fetchPage(ctx, p, q.URL, "")
		return nil
	})
	if err != nil {
		return SelectReport{}, err
	}
	if page.Page.Error != "" {
		return SelectReport{}, fmt.Errorf("reading %s: %s", q.URL, page.Page.Error)
	}
	// A stopped dev server or a missing page still renders a page with
	// text, so the browser's error page and HTTP errors are not read as ours.
	if u, err := url.Parse(page.Page.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return SelectReport{}, fmt.Errorf("%s did not load (browser ended on %s)", q.URL, page.Page.URL)
	}
	if page.Status >= 400 {
		return SelectReport{}, fmt.Errorf("%s returned HTTP %d", q.URL, page.Status)
	}
	if strings.TrimSpace(page.Text) == "" {
		return SelectReport{}, fmt.Errorf("no main text on %s", q.URL)
	}
	keyword := strings.TrimSpace(q.Keyword)
	lang := selectLanguage(q.Language, page.Page.Lang)
	words := contentWords(keyword)
	rep := SelectReport{
		URL: q.URL, FinalURL: page.Page.URL, Title: page.Page.Title, Language: lang, Words: len(strings.Fields(page.Text)),
		Keyword: keyword, Focus: focus(words, page.Page.Title, firstH1(page.Page.Headings)),
		About: []Selected{}, Mentions: []Selected{}, Dropped: []Dropped{}, Settings: q.Settings,
	}
	var notes []string
	max := r.LinkMaxBytes
	if max <= 0 {
		max = DefaultLinkMaxBytes
	}
	full := strings.TrimSpace(page.Page.Title + "\n\n" + page.Text)
	text := textrazor.Clip(full, max)
	if len(text) < len(full) {
		notes = append(notes, fmt.Sprintf("text cut to %d KB (textrazor.max_kb_per_page)", max/1024))
	}
	if n := focusNote(words, rep.Focus); n != "" {
		notes = append(notes, n)
	}

	progress(ctx, "linking the page's entities with TextRazor")
	res, hit, err := r.analyze(ctx, text)
	if err != nil {
		return SelectReport{}, err
	}
	rep.TextRazor.Cached = hit
	if !hit {
		rep.TextRazor.Units = res.Units
	}
	s := newSelection(res, page.Text, page.Links, page.Page.URL, words, q.Settings)

	grams := keywordGrams(keyword)
	var cands []*candidate
	var items map[string]wdEntity
	err = r.Browser.Use(ctx, func(p browser.Page) error {
		wr := &resolver{ctx: ctx, p: p, lang: lang}
		if err := wr.open(); err != nil {
			return err
		}
		progress(ctx, "looking the keyword up as Wikipedia titles")
		looks, hits, err := wr.keywordLookups(grams, s.known())
		if err != nil {
			return err
		}
		rep.KeywordLookups = append([]KeywordLookup{}, looks...)
		cands = s.candidates(hits, grams)
		qids := make([]string, len(cands))
		for i, c := range cands {
			qids[i] = c.QID
		}
		progress(ctx, fmt.Sprintf("checking %d entities by their Wikidata items", len(qids)))
		items, err = wr.articleItems(qids)
		return err
	})
	if err != nil {
		return SelectReport{}, err
	}
	rep.Dropped = append(rep.Dropped, s.dropped...)

	p9885 := map[string]string{}
	var kept, themes []*candidate
	for _, c := range cands {
		it, found := items[c.QID]
		id, reason := checkIdentity(it, found, lang, c.title, c.freebase)
		if reason != "" {
			rep.Dropped = append(rep.Dropped, c.drop(reason))
			continue
		}
		c.Name, c.Wikipedia = id.Title, string(schema.WikipediaLang(lang, id.Title))
		c.KGMID, c.GoogleKGID, c.KGMIDChecked, c.classes = id.KGMID, id.GoogleKGID, id.KGMIDChecked, id.Classes
		p9885[c.QID] = id.P9885
		if c.role == roleTheme {
			themes = append(themes, c)
		} else {
			kept = append(kept, c)
		}
	}
	themes, note := r.scoreThemes(ctx, keyword, text, themes, q.Settings, &rep)
	if note != "" {
		notes = append(notes, note)
	}
	kept = append(kept, themes...)

	about, mentions := roles(kept)
	var sel []*Selected
	for _, c := range slices.Concat(about, mentions) {
		sel = append(sel, &c.Selected)
	}
	switch {
	case r.Bing == nil:
		notes = append(notes, "no Bing IDs: set azure_language.endpoint and azure_language.api_key in the schema-mcp config file")
	case !azurelang.Supported(lang):
		notes = append(notes, "no Bing IDs: Azure entity linking reads only English and Spanish")
	default:
		ar, cached, err := r.bingLink(ctx, text, lang)
		if err != nil {
			notes = append(notes, "no Bing IDs: "+err.Error())
			break
		}
		rep.Azure = &AzureUsage{Cached: cached}
		if !cached {
			rep.Azure.Records = ar.Records
		}
		notes = append(notes, attachBing(sel, p9885, ar.Entities, lang)...)
	}
	// The node is built last, since the Bing ID is one of its identifiers.
	for _, c := range about {
		c.Thing = entityThing(lang, c.Selected, c.classes)
		rep.About = append(rep.About, c.Selected)
	}
	for _, c := range mentions {
		c.Thing = entityThing(lang, c.Selected, c.classes)
		rep.Mentions = append(rep.Mentions, c.Selected)
	}
	rep.Note = strings.Join(notes, "; ")
	return rep, nil
}

// check validates the query.
func (q SelectQuery) check() error {
	u, err := url.Parse(q.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("bad url %q: want an http or https URL", q.URL)
	}
	if strings.TrimSpace(q.Keyword) == "" {
		return errors.New("no keyword given")
	}
	s := q.Settings
	for _, v := range []struct {
		name string
		v    float64
	}{{"min_relevance", s.MinRelevance}, {"min_topic", s.MinTopic}, {"min_keyword_similarity", s.MinKeywordSimilarity}, {"min_page_similarity", s.MinPageSimilarity}} {
		if !(v.v >= 0 && v.v <= 1) {
			return fmt.Errorf("%s is %v: want 0 to 1", v.name, v.v)
		}
	}
	return nil
}

// selectLanguage is the edition asked for, else the page's lang, without
// its region. An edition that isn't a language code is en.
func selectLanguage(asked, page string) string {
	lang := strings.ToLower(strings.TrimSpace(cmp.Or(strings.TrimSpace(asked), page)))
	if i := strings.IndexAny(lang, "-_"); i >= 0 {
		lang = lang[:i]
	}
	if !languageCode.MatchString(lang) {
		return "en"
	}
	return lang
}

func firstH1(headings []string) string {
	for _, h := range headings {
		if text, ok := strings.CutPrefix(h, "h1: "); ok {
			return text
		}
	}
	return ""
}

// focusNote names the keyword words the title or the H1 lacks.
func focusNote(words []string, f map[string]FocusWord) string {
	var miss []string
	for _, w := range words {
		if !f[w].InTitle {
			miss = append(miss, `the title lacks "`+w+`"`)
		}
		if !f[w].InH1 {
			miss = append(miss, `the H1 lacks "`+w+`"`)
		}
	}
	if len(miss) == 0 {
		return ""
	}
	return strings.Join(miss, ", ") + ": the page's themes may describe its own topic rather than the keyword's"
}

// pageEntity is TextRazor's mentions of one Wikidata item on our page.
type pageEntity struct {
	title, qid, freebase string // TextRazor's entityId, wikidataId and first freebaseId
	rel, conf, topic     float64
	mentions             int
	matched, links       []string
	byScores, keyword    bool
}

// Roles a candidate is chosen for, in about order and then mentions.
const (
	roleKeyword = iota // a keyword page entity
	roleLookup         // a keyword lookup hit
	roleTheme
	roleMention
)

// candidate is an entity whose identity is still to be checked.
type candidate struct {
	Selected
	role     int
	title    string // the title its item must have; "" for a theme
	freebase string // TextRazor's Google ID for it
	label    string // a theme's TextRazor label, which is what is scored
	gram     int    // a lookup's place in the keyword
	classes  []string
}

func (c *candidate) drop(reason string) Dropped {
	return Dropped{Name: cmp.Or(c.Name, c.title, c.label), QID: c.QID, Source: c.Source, Relevance: c.Relevance, Topic: c.Topic,
		KeywordSimilarity: c.KeywordSimilarity, PageSimilarity: c.PageSimilarity, Reason: reason}
}

// selection is the page's entities and topics as TextRazor linked them.
type selection struct {
	set     SelectSettings
	text    string // the page's main text, without its title
	ents    []*pageEntity
	byQID   map[string]*pageEntity
	topics  map[string]textrazor.Topic // the best-scored topic per Wikidata item
	order   []string                   // topic items, first seen first
	dropped []Dropped                  // page entities failing the gate
}

// newSelection groups the mentions by Wikidata item, finds each one's
// own-site links and puts it through the gate. Mentions with no item
// (numbers, dates, e-mail addresses) are left out unlisted.
func newSelection(res textrazor.Result, text string, links []pageLink, finalURL string, words []string, set SelectSettings) *selection {
	s := &selection{set: set, text: text, byQID: map[string]*pageEntity{}, topics: map[string]textrazor.Topic{}}
	for _, t := range res.Topics {
		if t.WikidataID == "" {
			continue
		}
		best, ok := s.topics[t.WikidataID]
		if !ok {
			s.order = append(s.order, t.WikidataID)
		}
		if !ok || t.Score > best.Score {
			s.topics[t.WikidataID] = t
		}
	}
	for _, m := range res.Entities {
		if m.WikidataID == "" || m.ID == "" {
			continue
		}
		e := s.byQID[m.WikidataID]
		if e == nil {
			e = &pageEntity{title: m.ID, qid: m.WikidataID, topic: s.topics[m.WikidataID].Score, matched: []string{}}
			s.byQID[m.WikidataID] = e
			s.ents = append(s.ents, e)
		}
		e.mentions++
		e.rel, e.conf = math.Max(e.rel, m.Relevance), math.Max(e.conf, m.Confidence)
		e.freebase = cmp.Or(e.freebase, m.FreebaseID)
		if w := strings.Join(strings.Fields(m.MatchedText), " "); w != "" && !slices.ContainsFunc(e.matched, func(x string) bool { return strings.EqualFold(x, w) }) {
			e.matched = append(e.matched, w)
		}
	}
	page, _ := url.Parse(finalURL)
	for _, e := range s.ents {
		e.links = ownLinks(links, page, e.matched)
		e.byScores = e.rel >= set.MinRelevance && e.topic >= set.MinTopic
		e.keyword = e.byScores && sharesStem(e.title, words)
	}
	return s
}

// known is the keyword page entities, which the lookups count links to.
func (s *selection) known() []titleQID {
	var out []titleQID
	for _, e := range s.ents {
		if e.keyword {
			out = append(out, titleQID{e.title, e.qid})
		}
	}
	return out
}

// candidates turns the gated page entities, the lookup hits and the topics
// into one candidate per Wikidata item. A hit on a page entity's item moves
// that entity to the lookups, so no item is chosen twice. Page entities
// failing the gate, and would-be mentions not named in the main text, go to
// s.dropped.
func (s *selection) candidates(hits []lookupHit, grams []string) []*candidate {
	var out []*candidate
	hitQIDs := map[string]bool{}
	moved := map[*pageEntity]bool{}
	for _, h := range hits {
		hitQIDs[h.QID] = true
		c := &candidate{role: roleLookup, title: h.Title, gram: slices.Index(grams, h.Phrase),
			Selected: Selected{QID: h.QID, Source: SourceKeyword, Matched: []string{}, LinkedTo: []string{}}}
		if e := s.byQID[h.QID]; e != nil {
			moved[e] = true
			c.Selected, c.title, c.freebase = e.selected(SourceKeyword), e.title, e.freebase
		}
		c.QualifiedBy = []string{h.How}
		out = append(out, c)
	}
	for _, e := range s.ents {
		if moved[e] {
			continue
		}
		var by []string
		if e.byScores {
			by = append(by, "scores")
		}
		if len(e.links) > 0 {
			by = append(by, "link")
		}
		if len(by) == 0 {
			reason := ReasonBelowTopic
			if e.rel < s.set.MinRelevance {
				reason = ReasonBelowRelevance
			}
			s.dropped = append(s.dropped, e.drop(reason))
			continue
		}
		role := roleMention
		if e.keyword {
			role, by = roleKeyword, append(by, "keyword")
		} else if !slices.ContainsFunc(e.matched, func(f string) bool { return hasWords(s.text, f) }) {
			// Every mentions entity is named in the main text, not only in
			// the title TextRazor also read.
			s.dropped = append(s.dropped, e.drop(ReasonNotInPageText))
			continue
		}
		c := &candidate{role: role, title: e.title, freebase: e.freebase, Selected: e.selected(SourcePage)}
		c.QualifiedBy = by
		out = append(out, c)
	}
	for _, qid := range s.order {
		if s.byQID[qid] != nil || hitQIDs[qid] {
			continue
		}
		t := s.topics[qid]
		out = append(out, &candidate{role: roleTheme, label: t.Label, Selected: Selected{QID: qid, Source: SourceTheme, Topic: f64(t.Score),
			Matched: []string{}, LinkedTo: []string{}, QualifiedBy: []string{"similarity"}}})
	}
	return out
}

func (e *pageEntity) drop(reason string) Dropped {
	return Dropped{Name: e.title, QID: e.qid, Source: SourcePage, Relevance: f64(e.rel), Topic: f64(e.topic), Reason: reason}
}

func (e *pageEntity) selected(source string) Selected {
	return Selected{QID: e.qid, Source: source, Relevance: f64(e.rel), Topic: f64(e.topic), Confidence: f64(e.conf),
		Mentions: e.mentions, Matched: e.matched, LinkedTo: e.links}
}

func f64(v float64) *float64 { return &v }

// ownLinks returns the hrefs of links to other pages of our site, not its
// root, whose text holds one of forms as whole words, sorted. Hrefs that
// differ only in their fragment are one page, listed by the first href
// without a fragment, else the first seen.
func ownLinks(links []pageLink, page *url.URL, forms []string) []string {
	out := []string{}
	if page == nil {
		return out
	}
	at := map[string]int{} // a listed page, without its fragment, to its place in out
	site := func(u *url.URL) string { return strings.TrimPrefix(strings.ToLower(u.Host), "www.") }
	self := strings.TrimSuffix(page.Path, "/")
	for _, l := range links {
		u, err := url.Parse(l.Href)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || site(u) != site(page) {
			continue
		}
		if p := strings.TrimSuffix(u.Path, "/"); p == self || p == "" {
			continue
		}
		if !slices.ContainsFunc(forms, func(f string) bool { return hasWords(l.Text, f) }) {
			continue
		}
		key, _, frag := strings.Cut(l.Href, "#")
		if i, ok := at[key]; !ok {
			at[key] = len(out)
			out = append(out, l.Href)
		} else if !frag {
			out[i] = l.Href
		}
	}
	slices.Sort(out)
	return out
}

// hasWords reports whether text holds phrase as whole words, ignoring case.
func hasWords(text, phrase string) bool {
	text, phrase = strings.ToLower(text), strings.ToLower(phrase)
	if phrase == "" {
		return false
	}
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	for i := 0; i < len(text); {
		j := strings.Index(text[i:], phrase)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(phrase)
		before, _ := utf8.DecodeLastRuneInString(text[:start])
		after, _ := utf8.DecodeRuneInString(text[end:])
		if !isWord(before) && !isWord(after) {
			return true
		}
		_, n := utf8.DecodeRuneInString(text[start:])
		i = start + n
	}
	return false
}

// scoreThemes keeps the article themes whose labels are close to both the
// keyword and the page, comparing the unrounded cosines; the rest go to
// rep.Dropped. rep.Model is set only once scoring succeeded. The note says
// why there are no themes, if so.
func (r *Researcher) scoreThemes(ctx context.Context, keyword, page string, themes []*candidate, set SelectSettings, rep *SelectReport) ([]*candidate, string) {
	if r.Themes == nil {
		return nil, "no themes: no embedding model is set up"
	}
	if len(themes) == 0 {
		return nil, ""
	}
	labels := make([]string, len(themes))
	for i, c := range themes {
		labels[i] = c.label
	}
	progress(ctx, fmt.Sprintf("scoring %d themes against the keyword and the page", len(themes)))
	kw, pg, err := r.Themes.Score(ctx, keyword, page, labels)
	if err == nil && (len(kw) != len(labels) || len(pg) != len(labels)) {
		err = fmt.Errorf("%d and %d scores for %d themes", len(kw), len(pg), len(labels))
	}
	if err != nil {
		return nil, "no themes: " + err.Error()
	}
	rep.Model = r.Themes.Model()
	var kept []*candidate
	for i, c := range themes {
		c.KeywordSimilarity, c.PageSimilarity = f64(round3(kw[i])), f64(round3(pg[i]))
		if kw[i] >= set.MinKeywordSimilarity && pg[i] >= set.MinPageSimilarity {
			kept = append(kept, c)
		} else {
			rep.Dropped = append(rep.Dropped, c.drop(ReasonBelowSimilarity))
		}
	}
	return kept, ""
}

func round3(x float64) float64 { return math.Round(x*1000) / 1000 }

// roles orders the checked candidates. about is the keyword page entities
// by relevance, the lookup hits in keyword order, then the themes by topic
// score; mentions is the other page entities by relevance.
func roles(cands []*candidate) (about, mentions []*candidate) {
	byRelevance := func(a, b *candidate) int {
		return cmp.Or(cmp.Compare(*b.Relevance, *a.Relevance), cmp.Compare(*b.Topic, *a.Topic), cmp.Compare(b.Mentions, a.Mentions), strings.Compare(a.Name, b.Name))
	}
	group := func(role int) []*candidate {
		var out []*candidate
		for _, c := range cands {
			if c.role == role {
				out = append(out, c)
			}
		}
		return out
	}
	keyword, lookups, themes := group(roleKeyword), group(roleLookup), group(roleTheme)
	mentions = group(roleMention)
	slices.SortFunc(keyword, byRelevance)
	slices.SortStableFunc(lookups, func(a, b *candidate) int { return cmp.Compare(a.gram, b.gram) })
	slices.SortFunc(themes, func(a, b *candidate) int {
		return cmp.Or(cmp.Compare(*b.Topic, *a.Topic), strings.Compare(a.Name, b.Name))
	})
	slices.SortFunc(mentions, byRelevance)
	return append(append(keyword, lookups...), themes...), mentions
}

// bingLink links the text with Azure, or returns the cached result for the
// same language and text. A failed call is not cached.
func (r *Researcher) bingLink(ctx context.Context, text, lang string) (azurelang.Result, bool, error) {
	sum := sha256.Sum256([]byte(lang + "\n" + text))
	key := hex.EncodeToString(sum[:])
	var res azurelang.Result
	if r.Cache.Get("azure", key, pageTTL, &res) {
		return res, true, nil
	}
	progress(ctx, "linking the page's entities to Bing IDs with Azure AI Language")
	res, err := r.Bing.Link(ctx, text, lang)
	if err != nil {
		return azurelang.Result{}, false, err
	}
	r.Cache.Put("azure", key, res)
	return res, false, nil
}
