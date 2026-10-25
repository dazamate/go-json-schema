# select_entities keyword graph: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task by task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When the keyword has its own Wikipedia article, `select_entities`
builds `about` from that article's link graph and keeps only graph-linked
page entities in `mentions`.

**Architecture:** A new `internal/research/selection_graph.go` holds:
- choosing the centre from the keyword lookups;
- the centre's link neighbourhood (cached for 30 days);
- pruning by keyword similarity, the page's naming and the cap;
- counting links between page entities and graph members.

`SelectEntities` takes the graph path when a centre survives its checks,
and falls back to today's rules otherwise. The similarity comes from a new
`Similarity` method on the existing `ThemeScorer` (`embed.Scorer`).

**Tech stack:** Go, the existing browser resolver (Wikipedia and Wikidata
through ChromiumFish), `internal/embed` (bge-base ONNX), and the fake
Wikipedia in `entities_test.go`.

**Spec:**
- `docs/superpowers/specs/2026-10-25-keyword-graph-design.md` (the design);
- `docs/superpowers/specs/2026-10-25-own-page-entities-design.md` (the base
  spec, which the design amends).

## Global constraints

- No new module dependencies. `internal/research` must not import
  `internal/embed` outside `_test.go` files.
- All Wikipedia and Wikidata calls go through `resolver.fetch`. There are no
  search calls (`list=search`, `wbsearchentities`).
- Defaults, verbatim: `min_graph_similarity` 0.40, `min_graph_links` 3,
  `min_graph_links_ungated` 10, `min_mention_graph_links` 2,
  `max_graph_neighbours` 30.
- **Graph text:** `"<title>: <description>"`, or `"<title>"` when the
  description is empty. The keyword is embedded with
  `embed.QueryInstruction`, and texts are embedded alone.
- JSON names and identifiers use "centre" and "neighbour", as in the
  design (`centres`, `max_graph_neighbours`).
- The moving calculator golden lists (`TestSelectEntitiesCalibration`) are
  unchanged.
- Each task ends green on all three: `go vet ./...`,
  `CGO_ENABLED=0 go build ./...`, and offline `go test ./...`.
- Match the surrounding comment density and naming.
- Commits use Conventional Commits and end with
  `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`.

## Review focus

1. **A broad centre with thousands of links.** Every links page is
   followed, and the cap keeps at most 30 unnamed neighbours. Tests: Task 2
   (paging) and Task 5 (cap).
2. **A centre TextRazor also linked on the page.** It appears once, first,
   with `source` `keyword` and `centre` in `qualified_by`. Test: Task 7.
3. **A neighbour that is a list or category article.** It is dropped as
   `not_an_article` and doesn't count toward the 3-neighbour minimum.
   Test: Task 6.
4. **A second call within 30 days.** It reuses the cached neighbourhood and
   makes no link calls. Test: Task 2.
5. **A keyword with stop words** ("how to trade price action"). The centre
   comes from the 2-word gram "price action". Test: Task 4.

---

### Task 1: Similarity of graph texts to the keyword

**Files:**
- Modify: `internal/embed/embed.go`
- Test: `internal/embed/embed_test.go`
- Modify: `internal/research/serp.go` (the `ThemeScorer` interface)
- Modify: `internal/research/selection_test.go` (`fakeScorer`)

**Interfaces:**
- Produces:
  - `func (s *Scorer) Similarity(ctx context.Context, keyword string, texts []string) ([]float64, error)`;
  - `ThemeScorer` gains the same method, with its comment updated to
    "scores theme labels and graph texts";
  - `fakeScorer` gains `sim map[string]float64` (text to cosine) and
    `simTexts []string` (every text scored). Its `Similarity` returns
    `s.err` when set, and `fmt.Errorf("no similarity for %q", text)` for an
    unknown text.

- [ ] **Step 1: Write the failing tests.** Use `fakeEnc` (`vecs` maps each
  exact input to its vector, and `calls` records the inputs).

```go
func TestSimilarityUsesTheQueryInstruction(t *testing.T) {
	enc := &fakeEnc{vecs: map[string][]float64{QueryInstruction + "price action": {1, 0}, "Doji: Common pattern": {0.6, 0.8}, "Gap": {0, 1}}}
	s := &Scorer{Enc: enc}
	got, err := s.Similarity(context.Background(), "price action", []string{"Doji: Common pattern", "Gap"})
	// want got == [0.6, 0], and enc.calls == [QueryInstruction+"price action", "Doji: Common pattern", "Gap"]
}

func TestSimilarityOfNoTexts(t *testing.T) {
	// Similarity(ctx, "x", nil) returns nil, nil and encodes nothing.
}
```

- [ ] **Step 2:** Run `go test ./internal/embed -run Similarity`. Expected:
  FAIL (`Similarity` undefined).
- [ ] **Step 3:** Implement `Scorer.Similarity`. Add it to `ThemeScorer`
  and `fakeScorer`.
- [ ] **Step 4:** Run `go test ./internal/embed ./internal/research`.
  Expected: PASS.
- [ ] **Step 5:** Commit: `feat(embed): score graph texts against the keyword`.

### Task 2: The centre's link neighbourhood

**Files:**
- Create: `internal/research/selection_graph.go`
- Test: `internal/research/selection_graph_test.go`
- Modify: `internal/research/entities.go`. Add `LinksHere []struct{ Title string \`json:"title"\` } \`json:"linkshere"\`` to `wikiRespPage`.
- Modify: `internal/research/entities_test.go`:
  - `fakeWiki.linksHere map[string][]string`;
  - `linksJSON` lists every link of a page when `pltitles` is empty;
  - `prop=linkshere` answers from `linksHere`, paged by
    `linksPerResponse` with `lhcontinue`.

**Interfaces:**
- Produces:

```go
// neighbour is an article linked to or from the centre.
type neighbour struct {
	Title string `json:"title"`
	QID   string `json:"qid"`
	Desc  string `json:"desc,omitempty"` // Wikipedia's short description
	Out   bool   `json:"out,omitempty"`  // the centre links to it
	In    bool   `json:"in,omitempty"`   // it links to the centre
}

// graphHood is a centre's resolved neighbours and its raw link counts.
type graphHood struct {
	OutLinks   int         `json:"out_links"`
	InLinks    int         `json:"in_links"`
	InLinksCut bool        `json:"in_links_cut,omitempty"` // more than maxInLinks link here
	Neighbours []neighbour `json:"neighbours"`             // sorted by title
}

const maxInLinks = 500 // one linkshere page: the rest of a broad article's in-links are not read

func (r *resolver) graphHood(centre string) (graphHood, error)
func (r *Researcher) cachedHood(wr *resolver, centre string) (graphHood, error) // bucket "graph", key wr.lang+"|"+centre, TTL entityTTL
```

- **Requests:**
  - out-links: `action=query&prop=links&titles=<centre>&plnamespace=0&pllimit=max&redirects=1`;
  - in-links: `action=query&prop=linkshere&titles=<centre>&lhnamespace=0&lhshow=!redirect&lhlimit=max&redirects=1`.

  Follow `continue` on the out-links. On the in-links, stop once
  `maxInLinks` titles are read and set `InLinksCut` if a `continue` remains.
  Then resolve every title with `r.titles`.
- **What is skipped:** missing, invalid and disambiguation pages, pages with
  no `wikibase_item`, and the centre itself. Flags merge by resolved title.
- **Counts:** `OutLinks` and `InLinks` count the titles as listed, before
  resolving.

- [ ] **Step 1: Write the failing tests.**

```go
func TestGraphHoodLinksBothWays(t *testing.T) {
	w := &fakeWiki{
		pages: map[string]wikiPage{
			"Price action trading": {qid: "Q1"}, "Technical analysis": {qid: "Q2", desc: "Security analysis methodology"},
			"Gap (chart pattern)": {qid: "Q3", desc: "Price chart analysis concept"}, "Action": {qid: "Q8", disambig: true},
			"Orphan": {}, "Day trading": {qid: "Q6"}, "Chart pattern": {qid: "Q7"},
		},
		redirects:        map[string][2]string{"Gap pattern": {"Gap (chart pattern)", ""}},
		articleLinks:     map[string][]string{"Price action trading": {"Technical analysis", "Gap pattern", "Action", "Orphan", "Chart pattern", "Price action trading"}},
		linksHere:        map[string][]string{"Price action trading": {"Day trading", "Chart pattern"}},
		linksPerResponse: 2,
	}
	h, err := kwResolver(t, w).graphHood("Price action trading")
	// want h.OutLinks == 6, h.InLinks == 2, and h.Neighbours ==
	// [{Chart pattern Q7 "" Out In} {Day trading Q6 "" In} {Gap (chart pattern) Q3 "Price chart analysis concept" Out}
	//  {Technical analysis Q2 "Security analysis methodology" Out}]
}

func TestGraphHoodStopsAtMaxInLinks(t *testing.T) {
	// 501 titles L000..L500 link here, each with a QID; linksPerResponse 500.
	// h.InLinks == 500, h.InLinksCut, and len(h.Neighbours) == 500.
}

func TestCachedHoodFetchesOnce(t *testing.T) {
	// The same fake. r := &Researcher{Cache: NewCache(t.TempDir())}.
	// Call r.cachedHood twice: the second call adds 0 to w.fetches and
	// returns an equal graphHood.
}
```

- [ ] **Step 2:** Run `go test ./internal/research -run 'GraphHood|CachedHood'`.
  Expected: FAIL (undefined).
- [ ] **Step 3:** Implement the fake's support, `graphHood` and `cachedHood`.
- [ ] **Step 4:** Run `go test ./internal/research`. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(research): fetch a Wikipedia article's link neighbourhood`.

### Task 3: Distinct graph links between titles

**Files:**
- Modify: `internal/research/selection_keywords.go`. Rewrite `linkCounts`
  on top of `linkPairs`, keeping its behaviour.
- Modify: `internal/research/selection_graph.go`
- Test: `internal/research/selection_graph_test.go`

**Interfaces:**
- Produces:

```go
// linkPairs returns every {from, to} link between a title in a and one in
// b, in either direction: batched 50 by 50, following continue.
func (r *resolver) linkPairs(a, b []string) (map[[2]string]bool, error)

// linkedMembers returns, for each title, the sorted distinct members it
// links to or that link to it, itself excluded.
func (r *resolver) linkedMembers(titles, members []string) (map[string][]string, error)
```

Titles are canonical article titles, as `linkCounts` already assumes.

- [ ] **Step 1: Write the failing tests.**

```go
func TestLinkedMembersBothDirections(t *testing.T) {
	w := &fakeWiki{articleLinks: map[string][]string{
		"Foreign exchange market": {"Technical analysis", "Gap (chart pattern)"},
		"Chart pattern":           {"Foreign exchange market"},
		"Candle":                  {"Chart pattern"},
		"Technical analysis":      {"Technical analysis", "Gap (chart pattern)"},
	}}
	got, err := kwResolver(t, w).linkedMembers(
		[]string{"Foreign exchange market", "Candle", "Technical analysis"},
		[]string{"Technical analysis", "Gap (chart pattern)", "Chart pattern"})
	// want got["Foreign exchange market"] == [Chart pattern, Gap (chart pattern), Technical analysis],
	// got["Candle"] == [Chart pattern] and got["Technical analysis"] == [Gap (chart pattern)]
}

func TestLinkedMembersBatchesAndContinues(t *testing.T) {
	// 60 titles T00..T59, each linking to member M1; linksPerResponse 1.
	// Every title maps to [M1].
}
```

- [ ] **Step 2:** Run `go test ./internal/research -run LinkedMembers`.
  Expected: FAIL.
- [ ] **Step 3:** Implement `linkPairs` and `linkedMembers`, and port
  `linkCounts` onto `linkPairs`. The `TestLinkCounts*` and
  `TestKeywordLookups*` tests must stay green unchanged.
- [ ] **Step 4:** Run `go test ./internal/research`. Expected: PASS.
- [ ] **Step 5:** Commit: `refactor(research): share link pair queries between lookups and graph links`.

### Task 4: Choosing the centre

**Files:**
- Modify: `internal/research/selection_keywords.go`:
  - `KeywordLookup` gets an unexported `qid string`;
  - `keywordLookups` sets it for `exact` results and for every `redirect`
    result, link-confirmed or not.
- Modify: `internal/research/selection_graph.go`
- Test: `internal/research/selection_graph_test.go`

**Interfaces:**
- Produces:

```go
// centreCandidates returns the lookups that can be the keyword's centre,
// grouped by phrase length, longest first: exact or redirect results with
// a QID whose phrase has 2 or more words, or any word of a one-word
// keyword. A title appears once, in its longest group. Within a group,
// lookup order is kept.
func centreCandidates(looks []KeywordLookup, contentWords int) [][]titleQID
```

- [ ] **Step 1: Write the failing tests.** Use a table test,
  `TestCentreCandidates`. `lk(phrase, result, title, qid)` builds a
  `KeywordLookup`.

| Case | looks | contentWords | want |
|---|---|---|---|
| whole phrase | price exact Price Q8; price action redirect Price action trading Q1; price action trading exact Price action trading Q1; action disambiguation Action; action trading missing; trading redirect Trade Q5 | 3 | `[[{Price action trading Q1}]]` |
| moving | moving disambiguation; moving cost missing; moving cost calculator missing; cost exact Cost Q2; cost calculator missing; calculator exact Calculator Q3 | 3 | `nil` |
| one word | plumber exact Plumber Q4 | 1 | `[[{Plumber Q4}]]` |
| redirect, no confirmation | price exact Price Q8; price action redirect Price action trading Q1; price action strategy missing; strategy exact Strategy Q9 | 3 | `[[{Price action trading Q1}]]` |
| tie | foo bar exact A Q11; bar baz missing; baz qux exact B Q12 | 4 | `[[{A Q11} {B Q12}]]` |
| fallback group | x y z exact XYZ Q21; x y exact XY Q22; y z redirect XYZ Q21 | 3 | `[[{XYZ Q21}] [{XY Q22}]]` |
| stop words ("how to trade price action") | trade exact Trade Q5; trade price missing; trade price action missing; price exact Price Q8; price action redirect Price action trading Q1; action disambiguation | 3 | `[[{Price action trading Q1}]]` |

Also write `TestKeywordLookupsKeepCentreQIDs`, on a fake with "Price action
trading" exact (Q1) and "Price action" redirecting to it with no links. The
lookups for both carry `qid` "Q1", and `hits` still holds only the exact
hit.

- [ ] **Step 2:** Run `go test ./internal/research -run 'CentreCandidates|KeepCentreQIDs'`.
  Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run `go test ./internal/research`. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(research): choose the keyword's centre article`.

### Task 5: Graph settings, page naming and pruning

**Files:**
- Modify: `internal/research/selection.go`:
  - `SelectSettings`, `DefaultSelectSettings` and `check`;
  - the reason and source constants.
- Modify: `internal/research/selection_graph.go`
- Test: `internal/research/selection_graph_test.go` and
  `internal/research/selection_test.go` (the bad-query table and the
  settings echo).

**Interfaces:**
- Produces:

```go
// SelectSettings gains:
MinGraphSimilarity   float64 `json:"min_graph_similarity"`    // lowest neighbour-to-keyword cosine, for neighbours the page doesn't name
MinGraphLinks        int     `json:"min_graph_links"`         // graph members a gated page entity links with to join about
MinGraphLinksUngated int     `json:"min_graph_links_ungated"` // the same for a page entity failing the gate
MinMentionGraphLinks int     `json:"min_mention_graph_links"` // graph members a mention links with
MaxGraphNeighbours   int     `json:"max_graph_neighbours"`    // most kept neighbours the page doesn't name
// DefaultSelectSettings gains: MinGraphSimilarity: 0.40, MinGraphLinks: 3, MinGraphLinksUngated: 10, MinMentionGraphLinks: 2, MaxGraphNeighbours: 30

const SourceGraph = "graph" // an article linked to or from the keyword's centre
const (
	ReasonOffGraph             = "off_graph"              // too few links into the keyword's graph
	ReasonBelowGraphSimilarity = "below_graph_similarity" // a neighbour far from the keyword that the page doesn't name
)

// Dropped gains:
GraphSimilarity *float64 `json:"graph_similarity,omitempty"`
GraphLinks      *int     `json:"graph_links,omitempty"`

func graphText(n neighbour) string // "<title>: <desc>", or the title when desc is ""

// namedForms returns the page's spellings of title, without a trailing
// " (...)", in its plain form or with s or es: whole words, ignoring case,
// the first spelling of each form, in form order. A title of one word once
// the bracket goes returns nil: one-word titles are named only by TextRazor.
func namedForms(title, text string) []string

// graphNeighbour is a neighbour with what pruning found.
type graphNeighbour struct {
	neighbour
	Sim   *float64 // nil without a score
	Named []string // the page's spellings, or TextRazor's matched forms; nil when unnamed
}

type pruned struct {
	Kept    []graphNeighbour // by similarity, highest first, then the unscored named ones by title
	Dropped []Dropped        // scored neighbours cut: below_graph_similarity, source graph
	Cut     int              // len(Dropped) plus the neighbours left unscored
	Capped  int              // unnamed neighbours past max_graph_neighbours
}

// pruneNeighbours keeps the named neighbours, and the highest-scoring
// MaxGraphNeighbours unnamed ones at MinGraphSimilarity or more (compared
// unrounded). sims is by title; nil means there is no model.
func pruneNeighbours(ns []neighbour, sims map[string]float64, named map[string][]string, set SelectSettings) pruned
```

- **`check` errors**, verbatim:
  - `min_graph_similarity is %v: want 0 to 1`, which joins the existing 0
    to 1 list;
  - `min_graph_links is %d: want 1 or more`;
  - `min_graph_links_ungated is %d: want 1 or more`;
  - `min_mention_graph_links is %d: want 0 or more`;
  - `max_graph_neighbours is %d: want 0 or more`.
- **Ties in the cap:** equal scores order by title.
- **Dropped entries:** `Dropped.GraphSimilarity` is `round3` of the score.

- [ ] **Step 1: Write the failing tests.**

```go
func TestNamedForms(t *testing.T) {
	for _, c := range []struct{ title, text string; want []string }{
		{"Island reversal", "Two island reversals formed.", []string{"island reversals"}},
		{"Market reversal", "market reversals and a Market Reversal", []string{"Market Reversal", "market reversals"}},
		{"Head and shoulders (chart pattern)", "a head and shoulders top", []string{"head and shoulders"}},
		{"Bull market", "A bull marketing push", nil},
		{"Gap (chart pattern)", "Trade the Gap today.", nil}, // one word once the bracket goes
		{"Doji", "doji and DOJI", nil},                      // one-word titles are named only by TextRazor
	} { /* namedForms(c.title, c.text) == c.want */ }
}

func TestPruneKeepsNamedAndCuts(t *testing.T) {
	// ns A, B, C, D; sims {A: .6, B: .39, C: .30, D: .45}; named {B: ["b"]}; DefaultSelectSettings.
	// Kept titles [A, D, B]; Dropped [{C, below_graph_similarity, graph_similarity 0.3, source graph}];
	// Cut 1, Capped 0.
}

func TestPruneCapsUnnamed(t *testing.T) {
	// sims {A: .6, D: .45, E: .5, B: .2}; named {B}; MaxGraphNeighbours 2.
	// Kept [A, E, B]; Capped 1; D is not in Dropped.
}

func TestPruneWithoutModel(t *testing.T) {
	// sims nil; named {B}. Kept [B]; Dropped empty; Cut 3; Capped 0.
}

func TestPruneZeroCap(t *testing.T) {
	// MaxGraphNeighbours 0. Only B (named) is kept; Capped 2 (A and D pass 0.40).
}
```

  Add these entries to the `TestSelectEntitiesRejectsBadQueries` table:
  - `"min_graph_similarity is 1.5"`;
  - `"min_graph_links is 0: want 1 or more"`;
  - `"min_graph_links_ungated is 0: want 1 or more"`;
  - `"min_mention_graph_links is -1: want 0 or more"`;
  - `"max_graph_neighbours is -1: want 0 or more"`.

  Extend the settings echo in `TestSelectReportJSONKeys`:
  - `s["min_graph_similarity"] == 0.4`;
  - `s["max_graph_neighbours"] == float64(30)`.

- [ ] **Step 2:** Run `go test ./internal/research -run 'NamedForms|Prune|RejectsBadQueries|SelectReportJSONKeys'`.
  Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run `go test ./internal/research`. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(research): prune the keyword graph by score, naming and cap`.

### Task 6: Building the graph

**Files:**
- Modify: `internal/research/selection_graph.go`
- Test: `internal/research/selection_graph_test.go`. Add the shared
  fixture `patWiki()`, `patRaw()`, `patEntities()`, `patTopics()` and
  `patSims()`, described below.

**Interfaces:**
- Consumes:
  - `centreCandidates` (Task 4);
  - `cachedHood` (Task 2);
  - `pruneNeighbours`, `namedForms` and `graphText` (Task 5);
  - `linkedMembers` (Task 3);
  - `ThemeScorer.Similarity` (Task 1);
  - `checkIdentity` and `articleItems` (existing).
- Produces:

```go
// GraphReport is the keyword's graph as select_entities built it.
type GraphReport struct {
	Centres    []string `json:"centres"`
	OutLinks   int      `json:"out_links"`
	InLinks    int      `json:"in_links"`
	InLinksCut bool     `json:"in_links_cut,omitempty"`
	Neighbours int      `json:"neighbours"` // resolved articles linked to or from the centres
	Kept       int      `json:"kept"`       // neighbours kept after pruning and identity
	Cut        int      `json:"cut"`
	Capped     int      `json:"capped"`
}

// graphMember is a centre or a kept neighbour, its identity checked.
type graphMember struct {
	graphNeighbour
	Centre bool
	id     identity
}

type builtGraph struct {
	Members []graphMember       // centres in candidate order, then kept neighbours in pruned order
	Links   map[string][]string // a page entity or extra title -> the distinct members it links with
	Dropped []Dropped           // neighbours cut by score or failing identity
	Report  GraphReport
	Scored  bool // the model scored the neighbours
}

// buildGraph builds the keyword's graph from the centre groups. It returns
// nil and a note when no group's centre passes identity, or when fewer than
// 3 neighbours are kept after identity. extra is the lookup hits that
// TextRazor did not link on the page.
func (r *Researcher) buildGraph(ctx context.Context, wr *resolver, keyword string, s *selection, extra []titleQID, groups [][]titleQID, set SelectSettings) (*builtGraph, []string, error)
```

**Steps of `buildGraph`:**
1. Try each group in order. Run `articleItems` on its QIDs, then
   `checkIdentity(it, found, lang, title, "")`. The first group with a
   passing centre wins.
2. Run `cachedHood` for each winning centre, and merge the results:
   - neighbours union by title, with their flags ORed;
   - counts summed;
   - centres removed from the neighbours.
3. Build the `named` map, one entry per neighbour:
   - a page entity with the neighbour's QID gives its matched forms;
   - otherwise `namedForms(title, s.text)`.
4. Score the neighbours by similarity:
   - with `r.Themes`, run `r.Themes.Similarity(ctx, keyword, graphText for
     each neighbour)`;
   - with no model, use nil sims and the note `graph not scored: no
     embedding model is set up: only the neighbours the page names are
     kept`;
   - on a scoring error, use nil sims and the note `graph not scored: <err>:
     only the neighbours the page names are kept`.
5. Run `pruneNeighbours`.
6. Check the identity of each kept neighbour with
   `checkIdentity(it, found, lang, title, <page entity's freebase or "">)`.
   A failure goes to `Dropped` with its reason, and with source `page` if
   TextRazor linked it, else `graph`.
7. If fewer than 3 neighbours are kept, return nil and the note
   `<centres joined by " and "> has <n> kept neighbours (fewer than 3):
   about uses the page's themes`.
8. If every group fails identity, return nil and the note `no centre: <first
   centre> failed its identity check (<reason>): about uses the page's
   themes`.
9. Run `linkedMembers` on the titles of every page entity, plus the extra
   titles, against the member titles.
10. When `Cut + Capped > 0`, add the note `graph of <centres>: <Cut> cut
    below min_graph_similarity, <Capped> over max_graph_neighbours`.
11. When a centre's in-links were cut, add the note `graph of <centre>:
    only its first 500 in-links were read`.

**The pat fixture** (keyword `price action trading`, URL
`https://forex.example/price-action-trading/`):
- **Page:**
  - title `Price Action Trading`, lang `en`, H1 `Price action trading`;
  - text `Price action trading reads every candle. Technical analysis and
    forex go together. Island reversals are rare, so trade the gap.`
- **Wikipedia pages:**
  - Price action trading Q1;
  - Technical analysis Q2, "Security analysis methodology";
  - Gap (chart pattern) Q3, "Price chart analysis concept";
  - ISBN Q4, "Unique numeric book identifier";
  - Island reversal Q5, no description;
  - Day trading Q6, "Buying and selling within the same day";
  - Chart pattern Q7, "Pattern within a chart";
  - Action: disambiguation, with no links;
  - Orphan: no QID;
  - the redirect "Price action" → Price action trading;
  - the lookups "price", "action trading" and "trading" are missing.
- **Links:**
  - Price action trading links to Technical analysis, Gap (chart pattern),
    ISBN, Island reversal, Action and Orphan;
  - Day trading and Chart pattern link to Price action trading, in both
    `linksHere` and their own `articleLinks`;
  - Foreign exchange market links to Technical analysis and Gap (chart
    pattern);
  - Chart pattern links to Foreign exchange market;
  - Candle links to Chart pattern;
  - Trade links to Technical analysis and Day trading;
  - Swing trading links to Technical analysis, Gap (chart pattern) and
    Chart pattern;
  - Technical analysis links to Gap (chart pattern).
- **Wikidata items:** Q1 to Q7, plus Foreign exchange market Q10, Candle
  Q11, Trade Q12, Swing trading Q13 and Microeconomics Q14. Each has its
  sitelink.
- **TextRazor entities**, through `pageMention` (confidence 2):

| Entity | QID | Matched text | Relevance |
|---|---|---|---|
| Technical analysis | Q2 | "technical analysis" | 0.5 |
| Foreign exchange market | Q10 | "forex" | 0.7 |
| Candle | Q11 | "candle" | 0.58 |
| Trade | Q12 | "trade" | 0.87 |
| Swing trading | Q13 | "swing trading" | 0.12 |

- **Topics:** Technical analysis 0.9, Foreign exchange market 1, Candle 1,
  Trade 1, Microeconomics (Q14) 0.5.
- **Similarities**, by `graphText`:
  - "Technical analysis: Security analysis methodology" 0.42;
  - "Gap (chart pattern): Price chart analysis concept" 0.548;
  - "ISBN: Unique numeric book identifier" 0.299;
  - "Island reversal" 0.325;
  - "Day trading: Buying and selling within the same day" 0.576;
  - "Chart pattern: Pattern within a chart" 0.542.

- [ ] **Step 1: Write the failing tests.**

```go
func TestBuildGraphKeepsNamedAndScored(t *testing.T) {
	// Members titles: [Price action trading, Day trading, Gap (chart pattern), Chart pattern, Technical analysis, Island reversal].
	// Members[0].Centre; Gap's Named == nil (one word); Island reversal's Named == ["Island reversals"];
	// Technical analysis's Named == ["technical analysis"] (TextRazor's form).
	// Report == {Centres: [Price action trading], OutLinks: 6, InLinks: 2, Neighbours: 6, Kept: 5, Cut: 1, Capped: 0}.
	// Dropped == [{Name: ISBN, QID: Q4, Source: graph, GraphSimilarity: 0.299, Reason: below_graph_similarity}].
	// Links["Foreign exchange market"] == [Chart pattern, Gap (chart pattern), Technical analysis]; Links["Candle"] == [Chart pattern];
	// Links["Trade"] == [Day trading, Technical analysis]; Links["Swing trading"] == [Chart pattern, Gap (chart pattern), Technical analysis].
	// notes contain "graph of Price action trading: 1 cut below min_graph_similarity, 0 over max_graph_neighbours".
}

func TestBuildGraphDropsAListNeighbour(t *testing.T) {
	// Add out-link "List of chart patterns" (Q15, P31 Q13406463, sim .6). It is in Dropped as not_an_article with source graph,
	// is not a member, and Report.Kept stays 5.
}

func TestBuildGraphTriesTheNextCentre(t *testing.T) {
	// groups [[{XYZ Q21}], [{XY Q22}]]: Q21's sitelink is "Category:XYZ". The centre is XY.
}

func TestBuildGraphTooFewNeighbours(t *testing.T) {
	// No TextRazor entities, every similarity 0.1, and a text that names nothing: nil graph,
	// and a note containing "Price action trading has 0 kept neighbours (fewer than 3)".
}

func TestBuildGraphWithoutModel(t *testing.T) {
	// r.Themes == nil, and " A chart pattern helps." appended to the text.
	// Members: centre, Technical analysis, Island reversal and Chart pattern (the named ones);
	// Report.Cut 3; Dropped has no ISBN; a note contains "graph not scored: no embedding model is set up".
}
```

- [ ] **Step 2:** Run `go test ./internal/research -run BuildGraph`.
  Expected: FAIL.
- [ ] **Step 3:** Implement `buildGraph`.
- [ ] **Step 4:** Run `go test ./internal/research`. Expected: PASS.
- [ ] **Step 5:** Commit: `feat(research): build the keyword's entity graph`.

### Task 7: Graph roles in select_entities

**Files:**
- Modify: `internal/research/selection.go`:
  - `SelectEntities`;
  - `Selected` gains `GraphSimilarity *float64 \`json:"graph_similarity,omitempty"\`` and `GraphLinks int \`json:"graph_links,omitempty"\``;
  - `SelectReport` gains `Graph *GraphReport \`json:"graph,omitempty"\``;
  - `roles`, and the new role constants.
- Modify: `internal/research/selection_graph.go`. Add
  `(*selection).graphCandidates`.
- Test: `internal/research/selection_graph_test.go`,
  `internal/research/selection_golden_test.go` (one added assertion).

**Interfaces:**
- Consumes: `buildGraph` and `builtGraph` (Task 6), `centreCandidates`
  (Task 4).
- Produces:
  - roles `roleCentre`, `roleGraphPage`, `roleGraphLinked` and
    `roleNeighbour`;
  - `func (s *selection) graphCandidates(g *builtGraph, hits []lookupHit, set SelectSettings) []*candidate`,
    which appends drops to `s.dropped`.
  - Refactor the base gate's qualifiers into
    `func (e *pageEntity) qualifiers(set SelectSettings) []string`
    (`scores`, `link`), which both paths use.

**Flow in `SelectEntities`:** after `keywordLookups`, run
`centreCandidates(looks, len(words))`.
- With no groups, add the note `no Wikipedia article for the keyword: about
  uses the page's themes` and use the base path.
- Otherwise run `buildGraph`. Pass as `extra` the hits that are not centres
  and are not page entities.
- If it returns nil, add its notes and use the base path.
- Otherwise set `rep.Graph`, add its notes, use `graphCandidates`, and skip
  theme scoring. Set `rep.Model` to `r.Themes.Model()` when `g.Scored`.
- The identity loop, Bing and `thing` building stay shared.

**`graphCandidates`:**
- **Centre:** `roleCentre`, `qualified_by` `["centre"]`, `source`
  `keyword`. If TextRazor linked the same QID on the page, its `Selected`
  comes from that page entity (`e.selected(SourceKeyword)`), so it is
  listed once.
- **Member that is a page entity:** `roleGraphPage`, source `page`.
  - `qualified_by`: `out_link` and/or `in_link`, then `named`;
  - `graph_links` is `len(g.Links[title])`;
  - it skips the gate.
- **Other member:** `roleNeighbour`, source `graph`, with
  `graph_similarity` set and `matched` set to its `Named` forms
  (`[]` when none). `qualified_by` is the same as for a page member.
  `linked_to` is `[]`.
- **Any other page entity:** take `by := e.qualifiers(set)` and
  `n := len(g.Links[e.title])`.
  - **About:** `roleGraphLinked`, with `qualified_by` `by + "graph_links"`,
    when `len(by) > 0 && n >= MinGraphLinks` or
    `len(by) == 0 && n >= MinGraphLinksUngated`.
  - **Otherwise, when `len(by) > 0 && n >= MinMentionGraphLinks`:**
    - named in the main text: `roleMention`, with `qualified_by`
      `by + "graph_links"`;
    - not named: dropped as `not_in_page_text`.
  - **Otherwise:** dropped as `off_graph`, with `GraphLinks` set to `n`.
- **A lookup hit in `extra`:** `roleGraphLinked`, with `qualified_by`
  `[how, "graph_links"]`, when `n >= MinGraphLinksUngated`. Otherwise
  dropped as `off_graph`, source `keyword`.
- **Every theme whose QID isn't a member:** dropped as `off_graph`, source
  `theme`, with its topic score.

**Order:**
- **`about`:**
  1. centres, in candidate order;
  2. `roleGraphPage`, by `graph_links` descending, then relevance
     descending, then name;
  3. `roleGraphLinked`, ordered the same way;
  4. `roleNeighbour`, by `graph_similarity` descending, then name.
- **`mentions`:** as today.

- [ ] **Step 1: Write the failing tests** on the pat fixture, with
  `DefaultSelectSettings` unless noted.

```go
func TestSelectEntitiesGraphRoles(t *testing.T) {
	// about == [Price action trading, Technical analysis, Foreign exchange market, Day trading, Gap (chart pattern), Chart pattern, Island reversal]
	// Price action trading: source keyword, qualified_by [centre], no graph_links key in its JSON.
	// Technical analysis: source page, qualified_by [out_link named], graph_links 2 (Gap, and the centre's link to it).
	// Foreign exchange market: source page, qualified_by [scores graph_links], graph_links 3.
	// Day trading: source graph, qualified_by [in_link], graph_similarity 0.576, matched [].
	// Gap (chart pattern): source graph, qualified_by [out_link], matched [], no alternateName.
	// Island reversal: qualified_by [out_link named], graph_similarity 0.325, matched [Island reversals];
	// its thing's alternateName is ["Island reversals"].
	// rep.Model == "fake-bge": the model scored the graph.
	// mentions == [Trade], qualified_by [scores graph_links], graph_links 2.
	// dropped holds ISBN (below_graph_similarity, graph), Candle (off_graph, page, graph_links 1),
	// Swing trading (off_graph, page, graph_links 3) and Microeconomics (off_graph, theme).
	// rep.Graph == &GraphReport{Centres: [Price action trading], OutLinks: 6, InLinks: 2, Neighbours: 6, Kept: 5, Cut: 1, Capped: 0}.
	// No theme was scored: fakeScorer.labels is empty.
}

func TestSelectEntitiesGraphUngatedThreshold(t *testing.T) {
	// MinGraphLinksUngated 3: Swing trading joins about after Foreign exchange market (3 links each, lower relevance),
	// with qualified_by [graph_links].
}

func TestSelectEntitiesGraphCap(t *testing.T) {
	// MaxGraphNeighbours 1: Chart pattern is not in about or dropped; rep.Graph.Capped == 1.
}

func TestSelectEntitiesCentreLinkedOnThePage(t *testing.T) {
	// Add the TextRazor entity Price action trading Q1, "price action trading", relevance 0.9, topic 1.
	// about[0] is Price action trading, source keyword, with qualified_by containing centre; it appears once in about and mentions.
}

func TestSelectEntitiesGraphFallsBack(t *testing.T) {
	// Every similarity 0.1, and the text names no neighbour. Technical analysis is still kept (TextRazor names it), so 1 kept neighbour.
	// rep.Graph == nil; the note contains "fewer than 3"; the themes were scored (give fakeScorer.cos entries for each topic label).
}

func TestSelectEntitiesNoCentreNote(t *testing.T) {
	// The base selectFixture: the note contains "no Wikipedia article for the keyword: about uses the page's themes",
	// rep.Graph == nil, and the report JSON has no "graph" key.
}
```

  In `TestSelectEntitiesCalibration`, add: `rep.Graph == nil`, and
  `rep.Note` contains `no Wikipedia article for the keyword`. Its lists
  stay as they are.

- [ ] **Step 2:** Run `go test ./internal/research -run 'SelectEntitiesGraph|CentreLinked|NoCentreNote|Calibration'`.
  Expected: FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run `go test ./...`. Expected: PASS, with every existing
  select test unchanged.
- [ ] **Step 5:** Commit: `feat(research): select about and mentions from the keyword graph`.

### Task 8: Forex calibration fixtures and golden test

**Files:**
- Modify: `internal/research/linking.go`. Factor the key out of `analyze`
  as `func textrazorKey(text string) string`.
- Modify: `internal/research/selection_record_test.go`:
  - when `wikiRecorder.raw` is nil, `pageJS` passes through to the real
    page and is saved in `wikiRecorder.page`;
  - `prop=linkshere` is recorded into `LinksHere`;
  - `addPage` keeps `Description`.
- Modify: `internal/research/selection_golden_test.go`:
  - `fixturePage.Desc` (`desc,omitempty`);
  - `wikiFixture.LinksHere` (`links_here`);
  - `loadWikiFixture(t, dir string)` sets `wikiPage.desc` and
    `fakeWiki.linksHere`.

  Existing callers pass `"select"`. The moving `wiki.json` loads unchanged.
- Create: `internal/research/selection_forex_test.go`, with
  `TestRecordForexFixtures` and `TestSelectEntitiesForexCalibration`.
- Create: `internal/research/testdata/select-forex/page.json` (the
  `rawPage`), `textrazor.json` (a `textrazor.Result`), `wiki.json` and
  `cosines.json` (`{"keyword": ..., "texts": {text: cosine}}`).
- Modify: `docs/superpowers/specs/2026-10-25-keyword-graph-design.md`. Bring
  the Calibration section up to date with the recording.

**Interfaces:**
- Consumes: everything above, and `embed.NewONNX` and `embed.Scorer`
  (allowed in a `_test.go` file).

**Recording** (`TestRecordForexFixtures`, skipped unless
`SCHEMA_MCP_RECORD=1`):
- **Inputs:** URL `https://www.theforexguy.com/price-action-trading/`,
  keyword `price action trading`, and `DefaultSelectSettings`.
- **Browser:** the real browser through the recorder.
- **`Linker`:** a recording linker. It reads
  `NewCache(<os.UserCacheDir()>/go-json-schema).Get("textrazor", textrazorKey(text), pageTTL, &res)`.
  - When that misses and `TEXTRAZOR_API_KEY` is set, it calls
    `textrazor.Client{Key: os.Getenv("TEXTRAZOR_API_KEY")}`.
  - Otherwise it fails with `TextRazor answer not cached for this page
    text; set TEXTRAZOR_API_KEY`.
  - It saves the result it returns.
- **`Themes`:** a recording scorer around
  `embed.Scorer{Enc: embed.NewONNX(<user cache>/go-json-schema/models, Progress)}`,
  saving every `Similarity` text and score.
- **`Cache`:** `NewCache(t.TempDir())`, so every Wikipedia call is live and
  recorded.
- **Output:** the four files, with logs of `about`, `mentions`, `dropped`,
  `graph` and `note`.
- **Command:** `SCHEMA_MCP_RECORD=1 go test ./internal/research -run RecordForex -v -timeout 15m`.
  Never print or log the key.

**Golden** (`TestSelectEntitiesForexCalibration`): replay with
`selectPage(loadWikiFixture(t, "select-forex"), page, nil)`, a `fakeLinker`
returning `textrazor.json`, and a `fakeScorer` with `sim` from
`cosines.json`.
- `about` and `mentions` equal the recorded lists exactly. Write them into
  the test after recording.
- `rep.Graph` equals the recorded counts.
- `rep.Note` lacks `graph not scored`.
- These must hold, whatever the recording:
  - Foreign exchange market, Technical analysis, Market trend and
    Candlestick pattern are in `about`;
  - Candle, Candlestick and Signal are in `dropped` as `off_graph`;
  - neither ISBN nor One Direction is in `about` or `mentions`.
- The moving `TestSelectEntitiesCalibration` still passes.

- [ ] **Step 1:** Write `TestSelectEntitiesForexCalibration` with the
  must-hold assertions. It skips until `select-forex/wiki.json` exists, the
  same way `loadWikiFixture` skips.
- [ ] **Step 2:** Make the recorder and fixture changes. Run
  `go test ./internal/research`. Expected: PASS, with the forex golden
  skipped and the moving golden unchanged.
- [ ] **Step 3:** Record with the command above. If TextRazor misses the
  cache, stop and report NEEDS_CONTEXT. Do not look for the key.
- [ ] **Step 4:** Write the recorded `about` and `mentions` lists and graph
  counts into the golden test. Run `go test ./internal/research -run Calibration -v`.
  Expected: PASS.
- [ ] **Step 5:** Update the design doc's Calibration section to the
  recording. List every difference from the 2026-10-25 lists in the task
  report.
- [ ] **Step 6:** Commit: `test(research): record the forex calibration page for the keyword graph`.

### Task 9: MCP settings, tool description and docs

**Files:**
- Modify: `internal/mcpserver/server.go`:
  - `selectArgs` gains `MinGraphSimilarity *float64`, and `MinGraphLinks`,
    `MinGraphLinksUngated`, `MinMentionGraphLinks` and
    `MaxGraphNeighbours` as `*int`, with jsonschema text giving each
    default and range;
  - `settings()` applies them;
  - the `select_entities` description changes.
- Test: `internal/mcpserver/server_test.go`
- Modify: `README.md` (the `select_entities` section) and
  `internal/mcpserver/guide.md` (wherever it describes how `about` and
  `mentions` are chosen).
- Modify: `docs/superpowers/specs/2026-10-25-own-page-entities-design.md`.
  Fold the graph in: the signature, the inputs table, a Graph step in
  Steps, Output and reasons. Point to the keyword-graph design for the
  measurements.

**Description text to add**, after the about/mentions sentence:

> "When the keyword has its own Wikipedia article (the whole keyword, or a
> phrase of 2 or more of its words), about is instead that article, the
> articles linked to or from it that are close to the keyword or named on
> the page (at most max_graph_neighbours unnamed ones), and the page
> entities well linked into that graph; mentions then holds only page
> entities linked into it, and graph reports its size."

**Defaults sentence:** extend it with `min_graph_similarity 0.40,
min_graph_links 3, min_graph_links_ungated 10, min_mention_graph_links 2,
max_graph_neighbours 30`. Keep "our judgement, set by eye", and say the
cap was set by the owner.

- [ ] **Step 1: Write the failing test** `TestSelectEntitiesToolGraphSettings`:
  - `"max_graph_neighbours": -1` gives a tool error containing
    `max_graph_neighbours is -1: want 0 or more`;
  - `"min_graph_links": 0` gives one containing
    `min_graph_links is 0: want 1 or more`.
- [ ] **Step 2:** Run `go test ./internal/mcpserver -run GraphSettings`.
  Expected: FAIL (`unexpected additional properties`).
- [ ] **Step 3:** Implement the args, settings and description. Update the
  README, guide and base spec.
- [ ] **Step 4:** Run all of these. Expected: all clean.
  - `go vet ./...`
  - `CGO_ENABLED=0 go build ./...`
  - `go test ./...`
- [ ] **Step 5:** Commit: `feat(mcp): expose the keyword graph settings and document the graph`.
