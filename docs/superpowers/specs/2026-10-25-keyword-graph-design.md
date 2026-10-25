# select_entities: the keyword's entity graph

Date: 2026-10-25. Status: approved in conversation; this document records it.
It amends `docs/superpowers/specs/2026-10-25-own-page-entities-design.md`
(the base spec). Everything there still holds unless this document says
otherwise.

## Why

The second-page check ran `select_entities` on
https://www.theforexguy.com/price-action-trading/ with the keyword "price
action trading". It found two problems.

1. **`about` was full of general economics themes.** The 14 themes kept
   included Microeconomics, Macroeconomics, World economy, Corporate
   finance and Financial law. The owner called them "very general". They
   expected entities like the keyword itself: "price action trading",
   "price action", "forex trading".
2. **Wrong senses got into `mentions` on their scores:**
   - Candlestick (Q151771): "device used to hold a candle in place";
   - Candle (Q12888135): the wax candle;
   - Signal (Q174984): the physics sense.

   Their TextRazor confidences (2.0 to 2.6) are above those of correct
   calibration entities, so a confidence floor cannot remove them.

The owner's picture of `about` is an entity graph: the main keyword at the
centre, and every entity closely linked to it. The model pages are still
Digitaleer's and Salterra's (see the base spec).

## Goals

- When the keyword has its own Wikipedia article, `about` is that article
  plus its closely linked entities, taken from Wikipedia's link graph.
- Wrong senses drop out because they have no place in that graph.
- Pages whose keyword has no article of its own keep today's rule, so the
  approved moving calculator lists are unchanged.

## Non-goals and anti-goals

- **Our own site is never a source of graph edges.** The owner ruled this
  out explicitly. The graph comes only from Wikipedia and the keyword. Our
  page decides only which graph entities it names.
- No search results, no competitor pages, and no Wikidata or Wikipedia
  search by name. This is unchanged from the base spec.
- No paid entity-graph service: InLinks, Diffbot and WordLift were
  considered.
- No clickstream. It was tested and rejected (see Measures tried).
- No caps on `about` or `mentions`, except one: the owner capped the
  neighbours our page doesn't name at 30 (see Pruning). Broad articles
  otherwise flood `about`.

## When the graph applies: the centre

The keyword lookups already look up every word and run of adjacent words
in the keyword as an exact Wikipedia title (base spec, "Keyword
entities").

- The **centre** is the lookup hit whose phrase covers the most keyword
  words, provided its result is `exact` or `redirect` and either:
  - its phrase is the whole keyword, or
  - its phrase has 2 or more words.

  For example, "price action trading" is `exact`, so its centre is Price
  action trading (Q17051812). For "price action trading strategy", that
  same phrase would be the centre.
- If two hits tie for the most words, both are centres. *(Builder's call:
  their graphs are merged.)*
- A single word from a longer keyword is never a centre. The data shows why:
  - Price's and Trade's links lead into microeconomics;
  - Cost's lead into accounting;
  - Calculator's lead into calculator models;
  - Moving company's lead into real estate (Trulia, Case-Shiller index).
- A one-word keyword's own article is a centre, because it is the whole
  keyword.
- A `disambiguation` result is never a centre.
- **Redirects.** A `redirect` hit from a phrase of 2 or more words needs no
  link confirmation to be a centre. Wikipedia editors created that
  redirect for the whole phrase. Single-word redirects keep the base
  spec's confirmation rule. *(My call.)*
- **Identity.** The centre must pass the base spec's Identity checks. If
  it fails, the next hit by the same rule is tried (fewer words, still 2
  or more, or the whole keyword). If none passes, there is no centre.
  *(My call.)*
- **Too few neighbours.** A centre with fewer than 3 kept neighbours is
  treated as no centre, and `note` says why. A graph that small would
  leave `about` almost empty and `mentions` empty. *(My call.)*
- **No centre** (e.g. "moving cost calculator": "moving cost", "cost
  calculator" and the whole phrase are all `missing`): the base spec's
  rules apply unchanged, themes included.

## The graph

1. **Neighbours** are the main-namespace articles the centre links to
   (`prop=links`, `plnamespace=0`), plus those that link to it
   (`prop=linkshere`, `lhnamespace=0`, `lhshow=!redirect`), resolved
   through redirects.
   - Only the first 500 in-links are read, which is one page of
     `linkshere`. `graph.in_links_cut` and `note` say so when more exist.
   - Out-links are read in full. *(My call: on "technical analysis" the
     in-links held most of the companies, people and lists, and a broad
     article can have tens of thousands, each resolved and embedded
     before pruning.)*
   - Disambiguation pages and articles with no Wikidata item are skipped.
   - Pages that link only to one of the centre's redirects are not
     followed. *(My call: this matches the calibration run.)*
2. **Pruning.** Each neighbour is embedded as `"<title>: <short
   description>"` and compared with the keyword. The short description is
   Wikipedia's `prop=description`, which falls back to the Wikidata
   description. The keyword is embedded with the BGE query instruction,
   the same recipe as theme scoring (`internal/embed`).
   - A neighbour is **kept** if its similarity is ≥ `min_graph_similarity`
     (default **0.40**), **or** our page names it. The owner set this rule.
   - Otherwise it is cut.
   - **Cap.** At most `max_graph_neighbours` (default **30**, set by the
     owner) kept neighbours that our page does not name stay: the highest
     scoring. The rest are cut as `graph_cap`, counted in `graph.capped`
     and not listed. Neighbours our page names are never capped.
     - The cap was set after a broad test. For the keyword "technical
       analysis", the article has 801 neighbours and 748 pass at 0.40.
       They include companies (SoFi, Moomoo, Tehran Stock Exchange),
       people (Richard Wyckoff) and list articles.
     - Forex has 29 unnamed neighbours, so the cap doesn't touch it.
   - Our page is not otherwise scored against neighbours: the owner ruled
     out using our site to build the graph.
   - **No embedding model:** only the neighbours our page names are kept.
     The rest are counted in `graph.cut` but not listed, and `note` says
     they could not be scored. *(My call: no padding without a score.)*
3. **Named on the page** means either:
   - TextRazor linked the same Wikidata item on our page, at any score; or
   - the neighbour's title, without a trailing bracketed part, has 2 or
     more words and appears in the page's main text as whole words,
     ignoring case, in its plain form or with "s" or "es" added. For
     example, "Head and shoulders (chart pattern)" counts as "head and
     shoulders".

   One-word titles ("Gap (chart pattern)", "Stock", "Analysis") are named
   only through TextRazor. *(My call, after a broad check: against the
   Technical analysis article, single words and stripped brackets matched
   Analysis, Position (finance), Rally (stock market) and Market
   (economics) in the forex text, and named neighbours are never capped.)*
4. The **graph members** are the centre plus the kept neighbours. Each
   still goes through the base spec's Identity checks, so a list article
   or category page is dropped as `not_an_article`.

## Roles when there is a centre

- **`about`**, with no cap, in four groups in this order *(my call)*:
  1. **the centre or centres.** If TextRazor also linked a centre on the
     page, it keeps its `source` as today, and `qualified_by` gains
     `centre`;
  2. **page entities that are graph members**, by number of graph links,
     then relevance. They skip the base spec's gate;
  3. **page entities linked with enough graph members**, by number of
     graph links, then relevance. Links count in either direction,
     anywhere in either article, as distinct members. The threshold is:
     - `min_graph_links` (default **3**) if the entity passes the base
       spec's gate;
     - `min_graph_links_ungated` (default **10**) if it doesn't.
     *(My call; see Calibration for what each threshold admits.)*
  4. **the other kept neighbours**, by similarity.
- **`mentions`**: page entities that pass the base gate, are not in
  `about`, and are linked with at least `min_mention_graph_links`
  (default **2**) graph members. The base spec's rule that a mention must
  be named in the main text, not only the title, still applies.
- **Dropped:**
  - every other page entity, as `off_graph`;
  - every theme, as `off_graph`, since a theme is used only through the
    graph;
  - every keyword entity found by stems or lookups that is neither in the
    graph nor linked enough, as `off_graph`. For example Price and Trade,
    1 graph link each;
  - a cut neighbour, as `below_graph_similarity`, with its similarity.
- Every graph member and page entity still passes the base spec's
  Identity checks before it is kept.

## Settings

New settings, all our judgement from one page, like the others:

| Setting | Default | Meaning |
|---|---|---|
| `min_graph_similarity` | 0.40 | Lowest similarity to the keyword for a neighbour our page doesn't name, 0 to 1. |
| `min_graph_links` | 3 | Graph members a gated page entity must link with to join `about`, 1 or more. |
| `min_graph_links_ungated` | 10 | The same for a page entity that fails the gate, 1 or more. |
| `min_mention_graph_links` | 2 | Graph members a gated page entity must link with to stay in `mentions`, 0 or more. |
| `max_graph_neighbours` | 30 | Most kept neighbours our page doesn't name; set by the owner. 0 or more. |

The tool description and README must say these are judgement, as they do
for the existing cutoffs.

## Output changes

- **`source`:** `graph` for a kept neighbour our page doesn't link as a
  TextRazor entity.
- **`qualified_by`:**
  - the centre: `centre`;
  - a kept neighbour: `out_link` and/or `in_link`, plus `named` when our
    page names it;
  - a page entity that joined `about` or stayed in `mentions` by its
    links: `graph_links`.
- **`Selected`:** a new field `graph_links`, the number of distinct graph
  members the entity links with. It is omitted when there is no centre.
- **`dropped`:** reasons `off_graph` and `below_graph_similarity`, and a
  new field `graph_similarity`.
- **New top-level `graph`:**
  `{ centres: [title], out_links, in_links, in_links_cut?, neighbours, kept, cut, capped }`
  holds the centre titles and the counts. It is absent when there is no
  centre.
- **`note`:**
  - says when there is no centre ("no Wikipedia article for the keyword:
    about uses the page's themes");
  - says how many neighbours were cut.

## Calibration (2026-10-25)

Recorded on https://www.theforexguy.com/price-action-trading/ (2,406 words at the spike; 2,393 in the recording, which counts words by whitespace),
keyword "price action trading", with the defaults above. Price action
trading has 40 links out and 6 in. 35 neighbours pass at similarity ≥ 0.40,
and none of the low scorers is named on the page, so 9 are cut.

**`about` (42):**
- **Centre:** Price action trading.
- **Page entities in the graph:**
  - members: Technical analysis, Market trend, Candlestick pattern,
    Scalping (trading), Stock market index, Day trading;
  - by links: Stock (14), Stock market (13), Trader (finance) (13),
    Foreign exchange market (12), Swing trading (11), Broker (10).
- **Other neighbours:** Trading strategy, Algorithmic trading, Price
  channels, Order (exchange), Financial quote, Floor trader, Volatility
  (finance), Stock market index future, Gap (chart pattern), Chart
  pattern, Forex signal, Triangle (chart pattern), Open-high-low-close
  chart, Volume (finance), Commodity channel index, Broadening top, Market
  liquidity, Wedge pattern, Stock market data systems, Double top and
  double bottom, Support and resistance, Breakout (technical analysis),
  Head and shoulders (chart pattern), Candlestick chart, Trend line
  (technical analysis), Doji, Flag and pennant patterns, Slippage
  (finance), Moving average.

**`mentions`: none.** Every gated page entity is either in `about` or has
fewer than 2 graph links: Price 1, Trade 1, Candle 1, Candlestick 1,
Signal 0, Skill 0. The owner accepted an empty `mentions` here.

**Cut neighbours (`below_graph_similarity`):**

| Neighbour | Similarity |
|---|---|
| Triple top and triple bottom | 0.362 |
| Survivorship bias | 0.344 |
| Pattern formation | 0.331 |
| Hikkake pattern | 0.329 |
| Island reversal | 0.325 |
| ISBN | 0.299 |
| Extraordinary Popular Delusions and the Madness of Crowds | 0.298 |
| Failure rate | 0.281 |
| One Direction | 0.257 |

**What the link thresholds admit:**
- **Under `min_graph_links_ungated` 10 (left out):** Currency pair (4),
  Commodity (3), Currency (3), Economic bubble (3) and Investment
  management (3). These are mostly general finance terms the page mentions
  in passing (relevance 0.11 to 0.28).
- **At 3, they would join `about`.**
- **Below `min_graph_links` 3, all dropped `off_graph`:**
  - Real estate (2);
  - with 1 link: Bitcoin, Cryptocurrency, Money, Risk, United States
    dollar, Donald Trump, Statistics, Quality control and Trade idea.

**The moving calculator page has no centre,** so its golden lists are
unchanged.

The spike's data is in the session scratchpad and is not committed.
`TestRecordForexFixtures` (`selection_forex_test.go`) re-recorded the page on
2026-10-26 into `internal/research/testdata/select-forex`:
- `page.json`: our page's text and links (2,393 words, split on whitespace);
- `textrazor.json`: TextRazor's response, read from the cache of the
  2026-10-25 run, so the page text was unchanged and the recording cost 0
  units;
- `wiki.json`: the Wikipedia and Wikidata responses, trimmed to the fields
  read, including each page's short description and the in-links of the
  centre;
- `cosines.json`: the keyword's cosine against each of the 44 graph texts.

`TestSelectEntitiesForexCalibration` replays the recording offline and
asserts `about`, `mentions` and the `graph` counts exactly. The recording
agrees with the lists above: the same 42 `about` entities, no `mentions`, the
same 9 cut neighbours with the same similarities, and the same link counts
(40 out, 6 in, 44 neighbours, 35 kept, 0 capped). Only the order differs: the
lists above group `about` by role, while the report lists it in rank order,
with Stock, Trader (finance) and Stock market, then Foreign exchange market,
in that order, and Triangle (chart pattern) after Open-high-low-close chart.
- These hold in the golden test whatever the recording says:
  - Foreign exchange market, Technical analysis, Market trend and Candlestick
    pattern are in `about`;
  - Candle, Candlestick and Signal are dropped as `off_graph`;
  - neither ISBN nor One Direction is in `about` or `mentions`;
  - the moving golden lists are unchanged.

The repository is private, so storing theforexguy.com's text as a test
fixture is acceptable. *(My call.)*

## Measures tried and rejected, so the builder doesn't retry them

- **Absolute meaning check.** A "title: description" score against the
  page or the keyword cannot separate wrong senses from right ones:
  - wax Candle scores 0.278 against the keyword and 0.461 against the page;
  - FMCSA, a correct moving-page entity, scores 0.313 and 0.465.
- **Relative sense check.** This compares TextRazor's pick with the other
  senses on the word's disambiguation page.
  - It caught Candlestick: Candlestick chart fits the page 0.161 better.
  - It barely caught Signal: 0.099, against Signalling (economics).
  - It missed Candle: 0.028, against Candela.
  - It raised false alarms of 0.07: Truck against "Truck wages", and
    Trader (finance) against "Trader Monthly".
- **Requiring links in both directions.** Price action trading's article is
  short: only 2 of its links point back to it. Mutual links would also
  rank Foreign exchange market (2 mutual) with Economic bubble (2).
- **Clickstream** (Wikimedia's monthly reader-click counts).
  - Price action trading had 1.5k views and 60 outbound clicks in
    September 2026. Only 5 outbound and 2 inbound article pairs passed
    Wikimedia's more-than-10-clicks floor.
  - Pruning to them would leave 7 neighbours.
  - The data is a ~500 MB file each month.
- **Graphs from single keyword words.** See The centre.
- **Scoring neighbours against our page as well.** It was ruled out (our
  site is not a source). It was also unnecessary: dropping it added only
  Moving average.

## Cost

Per call with a centre, all through the browser like the other Wikipedia
calls:
- out-links: 1+ requests;
- in-links: 1+;
- descriptions and items for the neighbours: 1 per 50;
- link counting between page entities and graph members: about (page
  entities / 50) × (members / 50) requests in each direction.

The forex page needed about 15. No TextRazor or Azure units are added.
Wikipedia responses are cached for 30 days, like the entity lookups.

## Left to the builder

- Data structures, function boundaries and request batching.
- How to merge two tied centres' graphs.
- The order of the `qualified_by` values.
- Fixture trimming.
- Folding this document into the base spec once it is built, and updating
  the README, guide and tool description.
