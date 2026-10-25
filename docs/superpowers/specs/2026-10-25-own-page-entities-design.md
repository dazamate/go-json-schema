# Entity selection for a page and its keyword (`select_entities`)

Date: 2026-10-25. Status: draft, waiting for the owner's approval.

## Why

schema-mcp picks the `about` and `mentions` entities of a page's JSON-LD.
Until now they came from the ranking pages (`page_terms`) and from phrases
typed by hand, then looked up by name (`resolve_entities`). A test graph for
liftmyboxes.com/moving-cost-calculator/ built that way had 50 `mentions`,
138 aliases and 89 `additionalType` URLs. Many entities were loosely related
to the keyword, and some were wrong matches: "moving truck" resolved to
moving company and "staircase" to stairwell, both through Wikipedia
redirects nobody checked.

Google's general structured data guidelines
(https://developers.google.com/search/docs/appearance/structured-data/sd-policies)
say: "Don't mark up content that is not visible to readers of the page",
"Don't mark up irrelevant or misleading content, such as … content unrelated
to the focus of a page" and "Your structured data must be a true
representation of the page content." A manual action for this costs the
page its rich results; the same page says it "doesn't affect how the page
ranks in Google web search."

The owner's direction: entities must come from data, be tightly connected
to the target keyword, and must not depend on what competing sites do. We
assume competitors' schema is poor and aim to beat it, so search results
are not evidence for entity choice.

The first version of this design used only entities named on our page.
The owner found its `about` list too loose (price factors such as stairs
and weight). They pointed to Digitaleer's Fountain Hills SEO page and
salterrasite.com as models. Those fill `about` with the keyword's topic
family: its subject, location, parent field and sibling services. About
half of those entries are not named on the page, and neither page caps the
list (Digitaleer: 21 distinct `about` entries; Salterra: 27). The owner
chose that model after seeing the guideline quote above. So `about` now
holds the keyword's own entities and the page's themes, named on the page
or not, with no cap. `mentions` stays limited to what the page names.

## Goals

- A new MCP tool, `select_entities`, that reads **our own page** and its
  target keyword, and returns the entities for `about` and `mentions`, each
  as a ready schema.org node.
- `about` is the keyword's topic family, built like the owner's model pages.
  Every `mentions` entity is named in the page's visible text.
- Selection uses only measured data:
  - TextRazor's in-context entity linking and topics for our page;
  - our page's own links;
  - exact Wikipedia titles for the keyword's words, with Wikipedia's links
    between articles to settle disambiguation;
  - a local embedding model's similarity to the keyword and to the page.
- No fuzzy or search-based name lookup anywhere. The keyword's exact-title
  lookups are the one name-based step, under the rules in Keyword entities.
- Each entity node is lean: no alias dumps and no long type lists.
- Each entity node carries its Google, Bing, Wikipedia, Wikidata and
  DBpedia IDs, each found from data and cross-checked where a second source
  exists.

## Non-goals

- Search results, competitor pages and competitor schema play no part in
  entity choice.
- `knowsAbout` on the Organization. It is site-wide, not per keyword, and
  needs its own design.
- `keywords`, FAQ, HowTo, Dataset and the other non-entity parts of the
  graph. The stuffing guide's advice for them is unchanged here.
- `mentions` the page does not name. Only `about` may hold entities the
  page doesn't name. The editor can still add any node by hand, outside
  this tool.
- Reading text the caller supplies instead of a URL. Unpublished pages are
  read from the local dev server, which the browser can load.

## Anti-goals

These would make the feature a failure even if it ran:

- a `mentions` entity that the page's visible text does not name;
- an identity (Wikipedia title, QID, Google ID) found by a search or fuzzy
  name match. Identities come from TextRazor's links and topics, or from an
  exact Wikipedia title for a keyword word under the rules in Keyword
  entities;
- search-result data in the output or in the selection;
- a Wikipedia category, list or disambiguation page used as an entity.

## The owner's decisions

1. **Our page and its keyword are the only sources.** TextRazor reads our
   page's text. The keyword's words are looked up as exact Wikipedia titles.
2. **No competitor or search-result signal.** Not as a gate, not as a
   tie-break, not as output.
3. **`about` is the keyword's topic family, with no cap,** like Digitaleer's
   and Salterra's. It holds the keyword's own entities and the page's
   themes, named on the page or not. The owner chose this after seeing
   Google's guideline on marking up only visible content. An earlier cap of
   9 and an `about` list built from page relevance alone were both
   withdrawn.
4. **Identity is hardened** (see Identity).
5. **Padding is trimmed** (see The entity node).
6. **No search-result evidence in the output.** Each entity carries only
   our page's measurements and the reason it was kept or dropped.
7. **IDs: Google, Bing, Wikipedia, Wikidata and DBpedia.** The owner asked
   for these five after seeing what else exists (see Entity IDs). Bing IDs
   mean adding Azure AI Language.
8. **No caps** on `about` or `mentions`. The cutoffs alone decide the size.

The tool still reports when the page's title or H1 lacks a keyword word,
because the page's themes then describe the page's own topic, not the
keyword's.

## Calibration (one page, 2026-10-25)

Our page, read from the local server (`http://localhost:8080/moving-cost-calculator/`;
the live URL returned 404), main content only: 951 words, one TextRazor
request. 19 linked entities had a Wikidata ID. The data is in
`2026-10-25-own-page-entities/` next to this file:

- `page.html`: the page as served;
- `main.txt`: the text sent;
- `tr_page.json`: TextRazor's full response;
- `relatedness.json`: the rejected measures below, with each item's
  Wikidata label, description and enwiki sitelink;
- `kgmid_check.json`: TextRazor's `freebaseId` beside each item's Wikidata
  P646 and P2671. All 19 `freebaseId`s equal P646;
- `selection.json`: the full run of these rules: `about`, `mentions`,
  keyword lookups, identity drops and every theme that missed, with its
  similarities.

Page entities:

| Entity | Relevance | Topic score | Result with the defaults |
|---|---|---|---|
| Calculator Q31087 | 0.889 | 1.000 | about (keyword entity) |
| Moving company Q1757263 | 0.827 | 1.000 | about (keyword entity) |
| Cost Q240673 | 0.754 | 1.000 | about (keyword entity) |
| Truck Q43193 | 0.610 | 1.000 | mentions |
| Stairs Q12511 | 0.489 | 0.905 | mentions |
| ZIP Code Q136208 | 0.431 | 0.733 | mentions |
| Federal Motor Carrier Safety Administration Q5440305 | 0.413 | 0.616 | mentions |
| Weight Q25288 | 0.385 | 0.757 | mentions |
| Piano Q5994 | 0.372 | 0.687 | mentions |
| Hot tub Q1936429 | 0.319 | 0.591 | mentions |
| Billiard table Q3500725 | 0.130 | 0 | mentions (page links "Pool table moving cost" to `/cost-to-move-a-pool-table/`) |
| Email Q9158 | 0.262 | 0.135 | dropped |
| Car, Couch, GeoNames, Creative Commons license, Data, Nothing, Elevator | 0.174 or less | 0 | dropped |

Keyword lookups for "moving cost calculator":

- "cost" and "calculator" are exact titles, already found as page entities.
- "moving cost", "cost calculator" and the whole keyword have no article.
- "moving" is a disambiguation page with 30 options. Relocation (personal)
  and Relocation service each have 2 links to or from the keyword entities;
  no other option has more. Both go in `about`.

Themes: 69 TextRazor topics have a Wikidata ID. 17 are category pages or
have no English article. Six articles pass both cutoffs and go in `about`:

| Theme | Topic score | Similarity to keyword | Similarity to page |
|---|---|---|---|
| Transport | 0.375 | 0.404 | 0.576 |
| Freight transport | 0.320 | 0.400 | 0.595 |
| Business economics | 0.319 | 0.404 | 0.552 |
| Pricing | 0.275 | 0.461 | 0.643 |
| Land transport | 0.199 | 0.411 | 0.580 |
| Accounting | 0.134 | 0.409 | 0.552 |

The nearest misses, which the owner may want to bring in by loosening a cutoff:

| Theme | Topic score | Keyword | Page | Fails |
|---|---|---|---|---|
| Business | 0.458 | 0.355 | 0.514 | both |
| Computing | 0.333 | 0.409 | 0.492 | page |
| Supply chain management | 0.301 | 0.331 | 0.518 | both |
| Applied mathematics | 0.243 | 0.436 | 0.519 | page |
| Office equipment (article: Office supplies) | 0.229 | 0.408 | 0.541 | page |
| Mathematical tools (article: Mathematical instrument) | 0.205 | 0.489 | 0.541 | page |
| Financial accounting | 0.193 | 0.427 | 0.544 | page |

Result: `about` holds 11 (3 keyword page entities, 2 from the "moving"
lookup, 6 themes) and `mentions` 8. The sample built from it validates with
`build_jsonld`: 0 errors, 0 warnings.

- **Relevance** is TextRazor's `relevanceScore`: how on-topic the entity is for the text (0 to 1).
- **Topic score** is the `score` of the TextRazor topic with the same Wikidata ID, or 0 when there is none.

The topic score is where the clean break is: the top 10 score 0.591 or
more, Email 0.135 and the rest 0.

Measures tried and rejected, so the builder doesn't retry them:

- **Embedding similarity to the keyword alone, for ranking page entities**
  (bge-base-en-v1.5 on label, description and Wikipedia lead). Everything
  below the top three fell between 0.30 and 0.42, and Elevator (0.401) beat
  FMCSA (0.343). Similarity is used only as a filter on themes, and only
  combined with similarity to the page.
- **Wikipedia links for ranking relatedness.** Only Truck and FMCSA link to
  Moving company. Links are used only to settle a disambiguation page
  between a handful of options.
- **Wikipedia lead-section links as a topic family.** Each keyword word on
  its own drifts off the phrase: "cost" leads to accounting (cost of goods
  sold, income statements), "calculator" to hardware (graphing calculators,
  LCDs). The Moving company lead links only to a TV series, two cities and
  Germany. TextRazor's themes for the page read the whole phrase in context
  instead.
- **Exact label or alias search on Wikidata** for the keyword's words.
  "move" returned a family name first, and "calculator" ranked an
  unlabelled item above Q31087. Keyword lookups use exact Wikipedia titles
  instead, never Wikidata search.

TextRazor does not link the page's "move" or "moving". Relocation now comes
from the keyword lookup instead.

## Entity IDs (researched 2026-10-25)

What each source offers, checked against Wikidata's ID properties and the
services themselves. The data is in `2026-10-25-own-page-entities/`
(`id_properties.json`, `id_coverage.json`, `id_property_classes.json`).

| Source | Entity IDs | How we get them |
|---|---|---|
| Wikidata | The QID | TextRazor's `wikidataId`. |
| Wikipedia | The article title | The item's sitelink, checked against TextRazor's `entityId`. |
| DBpedia | A resource named after the English Wikipedia title, e.g. `http://dbpedia.org/resource/Piano` | Built from the checked English title, as `schema.Entity` does today. |
| Google | Freebase `/m/` MIDs (Wikidata P646) and Knowledge Graph `/g/` IDs (P2671) | From the item; `/m/` is cross-checked against TextRazor's `freebaseId`. All 11 calibration entities have a `/m/`, none a `/g/`. |
| Bing | Satori GUIDs, e.g. `a093e9b9-90f5-a3d5-c4b8-5855e1b01f85` (Wikidata P9885; 5,850 items have one, none of ours) | Azure AI Language entity linking returns `bingId` for each entity it links (https://learn.microsoft.com/en-us/azure/ai-services/language-service/entity-linking/quickstart?pivots=rest-api). Bing's own results page does not expose the ID. |
| DuckDuckGo | None: its entity panels are Wikipedia's (`"AbstractSource":"Wikipedia"` from api.duckduckgo.com) | Not used. |
| OpenAI / ChatGPT, Anthropic / Claude, Perplexity, Brave | None: no public entity IDs, and no Wikidata property names them | Not used. |
| Yandex | Wikidata P14153 ("Yandex Object Answer id") | Not asked for; none of ours have one. |
| 147 other Wikidata ID properties (library authorities, OpenAlex, Quora, Britannica and more) | 347 values across the 11 entities | Not asked for. |

Links checked on 2026-10-25 through ChromiumFish:

- `https://www.google.com/search?kgmid=/m/05r5c` opens Google's Piano
  knowledge panel.
- `https://www.google.com/search?q=Piano&kponly&kgmid=/m/05r5c`, the form
  `schema.KGSearch` builds today, returns "did not match any documents" and
  no panel.
- `https://g.co/kg/m/05r5c`, Wikidata's P646 link format, shows no panel either.
- Wikidata's Bing link format,
  `https://www.bing.com/search?FORM=SNAPST&q=*&filters=sid%3a%22<id>%22`, now
  searches for a literal "*". No working Bing link exists, so the Bing ID
  goes in `identifier` only.

Azure AI Language limits that shape the design
(https://learn.microsoft.com/en-us/azure/ai-services/language-service/concepts/data-limits):

- At most 5,120 characters per document and 5 documents per request in a
  synchronous call.
- Billing is per text record of 1,000 characters; a free F0 tier exists.
- Entity linking retires on 1 September 2028
  (https://learn.microsoft.com/en-us/azure/ai-services/language-service/entity-linking/how-to/call-api).
  After that, Bing IDs stop and everything else keeps working.

## The tool

`select_entities(url, keyword, language?, min_relevance?, min_topic?, min_keyword_similarity?, min_page_similarity?, min_link_confidence?, min_graph_similarity?, min_graph_links?, min_graph_links_ungated?, min_mention_graph_links?, max_graph_neighbours?)`

| Input | Default | Meaning |
|---|---|---|
| `url` | required | Our page. Any URL the browser can load, including `localhost`. |
| `keyword` | required | The page's target keyword: looked up for keyword entities, compared with themes, and used for the focus check. |
| `language` | the page's `lang`, else `en` | Wikipedia edition for titles and sitelinks. |
| `min_relevance` | 0.3 | Lowest TextRazor relevance for a page entity to qualify. |
| `min_topic` | 0.5 | Lowest topic score for a page entity to qualify. |
| `min_keyword_similarity` | 0.40 | Lowest similarity between a theme and the keyword. |
| `min_page_similarity` | 0.55 | Lowest similarity between a theme and the page. |
| `min_link_confidence` | 1.5 | Lowest TextRazor confidence (about 0.5 to 10, so 0 or more rather than 0 to 1) for a page entity to qualify through an own-site link. |
| `min_graph_similarity` | 0.40 | Lowest similarity between a linked article and the keyword, 0 to 1, for an article the page does not name. |
| `min_graph_links` | 3 | Links into the graph a page entity that qualifies on its own (relevance, topic score or own-site link) needs to join `about`. 1 or more. |
| `min_graph_links_ungated` | 10 | Links into the graph a page entity that does not qualify on its own needs to join `about`; also the count a keyword lookup hit TextRazor did not link must reach. 1 or more. |
| `min_mention_graph_links` | 2 | Links into the graph a page entity that qualifies on its own needs to stay in `mentions`; it must also be named in the page's main text, or it is dropped as `not_in_page_text`. 0 or more. |
| `max_graph_neighbours` | 30 | Most unnamed linked articles that join `about`. 0 or more; set by the owner. |

The defaults above are our judgement, set by eye from the one calibration
page above, not from any Google source. The tool description and README
must say so, and they need checking on a second page. Without a TextRazor
key the tool fails with the same message `page_terms` gives
(`noLinkerNote`).

### Steps

1. **Read the page** through the ChromiumFish browser with the same
   main-content logic as `page_terms` (`pageJS` in `internal/research/pages.go`).
   Also collect every link in the main content: its href and visible text.
2. **Focus check:** for each keyword word, report whether it appears in the
   title and in the H1, matching word stems. Report only; never block.
3. **Link the entities:** send the title, a newline and the main text to
   TextRazor (`extractors=entities,topics`). Cut the text to the configured
   `max_kb_per_page`. Cache by the text's SHA-256 in the `textrazor` bucket,
   as `linking.go` does.
4. **Group** the mentions by Wikidata ID. Drop mentions with no Wikidata ID
   (numbers, dates, e-mail addresses), without listing them. For each
   entity keep:
   - its relevance and confidence: the highest over its mentions;
   - its topic score;
   - its mention count;
   - its matched forms (`matchedText`), deduplicated ignoring case and kept
     in first-seen spelling;
   - its own-site links: main-content links to the page's own host whose
     visible text contains a matched form as whole words, ignoring case.
5. **Gate:** a page entity qualifies on one of two paths:
   - its scores: relevance ≥ `min_relevance` **and** topic score ≥ `min_topic`;
   - an own-site link **and** confidence ≥ `min_link_confidence`. A link
     makes no relevance check, so this floor keeps out TextRazor's unsure
     links. Added after the calibration run: on the page as served without
     its CSS, "Slate" in "Slate or wood", on a card linking to the pool
     table page, was linked to Slate (magazine) at confidence 1.08 and
     relevance 0.057. The link-only entities kept there were Billiard table
     (1.93; 1.74 in the calibration fixture), Mobile home (2.19) and
     Apartment (2.85). An entity whose link fails the floor but whose
     scores pass is kept, without `link` in `qualified_by`.
6. **Keyword entities** (see Keyword entities):
   - qualifying page entities whose Wikipedia title shares a word stem with
     the keyword;
   - the keyword's words and phrases looked up as exact Wikipedia titles.
7. **Themes** (see Themes): TextRazor's topics for the page that are real
   articles and pass both similarity cutoffs.
8. **Identity** (below) for every candidate. A candidate that fails is
   dropped with its reason. Then, when an Azure AI Language key is
   configured, add Bing IDs (see Bing ID).
9. **Roles**, with no caps:
   - `about`: the keyword entities, then the themes.
     - Keyword page entities come first, by relevance.
     - Then the lookup results, in keyword order.
     - Then the themes, by topic score.
   - `mentions`: every other qualifying page entity, by relevance, then
     topic score, then mention count, then name. This includes entities
     that qualified only through a link.
10. **Graph** (see [the keyword graph design](2026-10-25-keyword-graph-design.md),
    which holds the measurements). When the keyword has its own Wikipedia
    article (the whole keyword, or a phrase of 2 or more of its words), build
    the graph around it: the article, the articles linked to or from it, and
    the page entities linked into it. `about` is then the centre articles, the
    neighbours that are close to the keyword (`min_graph_similarity`) or named
    on the page (at most `max_graph_neighbours` unnamed ones), and the page
    entities with enough links into the graph: `min_graph_links` for one that
    qualifies on its own (step 5), `min_graph_links_ungated` for one that
    does not and for a keyword lookup hit TextRazor did not link.
    `mentions` keeps only page entities that qualify on their own, are named
    in the page text and have `min_mention_graph_links` links. With no article, or fewer than 3 kept
    neighbours, steps 6, 7 and 9 stand and `note` says so.
11. **Build each entity's node** (below).

### Keyword entities

- **From the page:** a qualifying page entity whose Wikipedia title shares
  a word stem with a keyword word ("Moving company" for "moving").
- **From exact titles:** look up every word and run of adjacent words in
  the keyword as a Wikipedia title (`action=query`, `redirects=1`,
  `prop=pageprops` for `wikibase_item` and `disambiguation`).
  - An **exact** title with a QID that isn't already found is a keyword
    entity.
  - A **missing** title is skipped.
  - A **disambiguation** page is settled by links:
    - count, for each option on the page, its links to the keyword entities
      already found plus their links to it, over whole articles
      (`prop=links` with `pltitles`);
    - the options with the highest count go in, ties included, when that
      count is at least 1;
    - with no links at all, nothing goes in.
  - A **redirect** is settled the same way as a single option. It goes in
    only when it links to, or is linked from, a keyword entity.
- The step is the one name-based lookup in the tool. It uses exact titles
  only, never Wikipedia or Wikidata search.

### Themes

- Candidates are TextRazor's topics for the page with a Wikidata ID, not
  already a page entity or a keyword entity.
- A theme must be a real article. Drop it when its item has no sitelink in
  the language's Wikipedia, or when it is a category, list or
  disambiguation item (see Identity). Topic labels are category-style
  ("Computers", "Office equipment"), so the node's `name` is the sitelink
  title ("Computer", "Office supplies"), never the label.
- A theme goes in `about` when its similarity to the keyword is at least
  `min_keyword_similarity` and its similarity to the page is at least
  `min_page_similarity`.
- Every theme that misses is listed in `dropped` with both similarities, so
  the owner can see what a looser cutoff would add.

### Similarity

The cutoffs only mean something with the exact recipe they were set with,
which is google-seo-docs' (`seoaudit/embed.py`):

- the model is BAAI/bge-base-en-v1.5, the quantized ONNX build fastembed
  uses;
- the keyword is embedded as a query, with BGE's instruction "Represent
  this sentence for searching relevant passages: ";
- the page is the main text cut into 200-word windows that overlap by 50
  words (at most 40). Its vector is the normalised mean of the windows;
- a theme is embedded by its TextRazor label alone, as a passage with no
  instruction;
- similarity is the cosine of unit vectors.

The model runs locally, inside schema-mcp. No text goes to an outside
embedding API. The builder picks the Go binding and where the model file is
cached (about 210 MB, downloaded on first use like google-seo-docs does).
Without the model the tool still runs, with no themes, and `note` says why.

### Identity

- The identity is a QID: TextRazor's `wikidataId` for a page entity or a
  theme, or the `wikibase_item` of an exact title for a keyword lookup.
  It is never a name.
- Fetch each item by QID from Wikidata (`wbgetentities`) inside the browser,
  as the resolver does, with sitelinks for the language's wiki, the English
  label, and claims P31, P279, P646 and P2671.
- **Drop the entity as `identity_mismatch`** when the item's sitelink title
  differs from the title it was found by: TextRazor's `entityId` for a page
  entity, or the looked-up title for a keyword entity. Compare with spaces
  for underscores and the first letter case-folded. A theme has no title to
  compare, so its sitelink becomes its name. Any entity is also dropped
  when its item has no sitelink in that language.
- **Drop it as `not_an_article`** when the item is a disambiguation page
  (P31 Q4167410), a Wikimedia category (Q4167836) or a list (Q13406463).
- **Google IDs:** the item's own P646 (`/m/`) and its own P2671 (`/g/`),
  both when both exist. The synonym borrowing `resolve_entities` does
  (`synonymProps`) is not used here. A missing Google ID is left out, never
  filled.
- **Google ID cross-check:** TextRazor gives each linked entity a
  `freebaseId` (`/m/`) from its own knowledge base, which is a second,
  independent source for the same ID.
  - When TextRazor's `freebaseId` and the item's P646 both exist and
    differ, drop the entity as `identity_mismatch`.
  - When they agree, set `kgmid_checked: true` on the entity.
  - When either one is missing, keep the entity with `kgmid_checked: false`.
    The Google ID still comes only from Wikidata, so a `freebaseId` that
    Wikidata lacks is never used.
  - A `/g/` ID (P2671) can't be cross-checked, because TextRazor gives only
    `/m/` IDs.
  - Themes and keyword lookups have no TextRazor `freebaseId`, so they
    always have `kgmid_checked: false`.

### Bing ID

- Send the same text to Azure AI Language entity linking
  (`kind: EntityLinking`, `analyze-text`). Split it at paragraph breaks into
  documents of at most 5,120 characters, with up to 5 documents per request.
- Azure's result gives each linked entity a `bingId` and a Wikipedia `url`.
  A selected entity takes the `bingId` whose `url` is the entity's own
  Wikipedia article (same title, compared as for the sitelink).
  - Azure's title must match exactly. A near match is never used.
  - An entity Azure did not link on our page gets no Bing ID. It is still
    selected, with `bing_id` empty.
- **Cross-check:** when the Wikidata item has a P9885 value and it differs
  from Azure's `bingId`, leave the Bing ID out and say so in `note`. The
  entity stays, since its other IDs passed. When they agree, set
  `bing_checked: true`.
- Cache Azure results by the text's SHA-256, like TextRazor's.
- Without Azure settings in the config file, the tool runs without Bing IDs
  and `note` says why.
- Settings go in `schema-mcp.yml` under `azure_language` (`endpoint`,
  `api_key`), with `schema-mcp.yml.dist` updated to match. The key is a
  secret like TextRazor's: never logged, never echoed in errors (as
  `TestAnalyzeErrorNeverEchoesTheKey` does for TextRazor). Azure also gets
  only text the browser already fetched.

### The entity node

A `Thing` built with `schema.Entity` (`schema/entity.go`). The IDs are the
owner's five families, and no others:

- `name`: the Wikipedia article title.
- `sameAs`:
  - the Wikipedia article URL;
  - the Wikidata item URL;
  - `https://www.google.com/search?kgmid=<id>` for each Google ID (`/m/`
    and `/g/`);
  - the DBpedia resource, when the page is in English (DBpedia's main
    edition follows English Wikipedia titles).
  - No DuckDuckGo or Bing link: DuckDuckGo was not asked for, and no
    working Bing link exists.
- `identifier`, as `PropertyValue`s:
  - the Wikidata QID;
  - each Google ID, with the existing GoogleMachineID shape;
  - the Bing ID, when Azure gave one.
  - The builder picks the `propertyID` names, keeping GoogleMachineID as it
    is today.
- `alternateName`: the matched forms our page used, minus any equal to
  `name` ignoring case. Nothing from Wikidata aliases. Omitted when empty,
  which is always the case for themes and keyword lookups.
- `additionalType`: at most 2 Wikidata class URLs. Take P31 values first and
  fill with P279 values, each in claim order with preferred-rank values
  first. Omitted when empty.
- No `@id`. The caller or the editor assigns one.

### Output

```
{
  url, final_url, title, language, words,
  keyword, focus: { word: { in_title, in_h1 } },
  about:    [ Selected ],
  mentions: [ Selected ],
  keyword_lookups: [ { phrase, result, title?, chosen? [], links? } ],
  dropped:  [ { name, qid, source, relevance?, topic?, confidence?,
                keyword_similarity?, page_similarity?, reason } ],
  graph: { centres, out_links, in_links, in_links_cut, neighbours, kept,
           cut, capped }?,
  settings: { min_relevance, min_topic, min_keyword_similarity, min_page_similarity,
              min_link_confidence, min_graph_similarity, min_graph_links,
              min_graph_links_ungated, min_mention_graph_links,
              max_graph_neighbours },
  textrazor: { units, cached },
  azure: { records, cached }?,
  model,
  note
}
Selected = { name, qid, source, wikipedia, kgmid?, google_kg_id?,
             kgmid_checked, bing_id?, bing_checked, relevance?, topic?,
             confidence?, keyword_similarity?, page_similarity?,
             graph_similarity?, graph_links?, mentions?, matched [], linked_to [], qualified_by, thing }
```

- `source` is `page`, `keyword`, `theme` or `graph`.
- `qualified_by`:
  - for a page entity: `scores`, `link` or both, plus `keyword` when it is
    a keyword entity;
  - for a lookup: `exact`, `disambiguation` or `redirect`;
  - for a theme: `similarity`.
- `keyword_lookups` records each word or phrase looked up: `exact`,
  `missing`, `disambiguation` or `redirect`, with the options chosen and
  their link counts.
- `reason` is `below_relevance`, `below_topic`, `below_confidence` (an
  own-site link under `min_link_confidence` and scores under the cutoffs),
  `below_similarity`, `off_graph` (a page entity with too few links into the
  graph), `below_graph_similarity` (a linked article under
  `min_graph_similarity`), `not_in_page_text` (a would-be mention named only in
  the title), `identity_mismatch` or `not_an_article`.
- `model` names the embedding model that ran.
- `note` explains a failed focus check, a cut text, or a missing model or
  key.

## Changes to existing parts

- **`internal/mcpserver/server.go` instructions.** Entities for `about` and
  `mentions` come from `select_entities` on the target page. `page_terms`
  entities and `resolve_entities` lookups no longer feed them.
- **The `entity_schema` prompt.** Add a `page_url` argument. Replace steps
  3–4 with one call to `select_entities` that uses its `about` and
  `mentions` as given.
- **`internal/mcpserver/guide.md`.** Rewrite the `about`, `mentions`,
  `alternateName` and `additionalType` rows and their volume targets to the
  rules above. Its comment says it is adapted from a recipe pipeline guide
  ("Update both together"); this change is for schema-mcp only, so the
  comment must say the two now differ on these rows. Other rows are unchanged.
- **README.** Add the tool to the tools table and a section on the rules,
  the defaults' source and the calibration.
- **`schema.KGSearch` (`schema/entity.go`).** It builds
  `https://www.google.com/search?q=<name>&kponly&kgmid=<mid>`, which no
  longer shows a panel. Change it to `https://www.google.com/search?kgmid=<mid>`.
  This also changes the links `resolve_entities` and every other caller
  emit. Update the guide's "URL sources" line to match.
- **`cmd/schema-mcp/config.go`.** Add an `azure_language` section
  (`endpoint`, `api_key`), and add it to `schema-mcp.yml.dist` with empty
  values. Unknown keys stay errors.
- **New client** for Azure AI Language entity linking, next to
  `internal/textrazor`.
- **New embedding package** that runs bge-base-en-v1.5 locally with the
  recipe in Similarity, behind an interface the tests can fake.
- **`guide.md` volume targets:** `about` and `mentions` have no targets.
  The tool decides their size.

`page_terms`, `resolve_entities` and `serp` keep working as they are.

## Tests

- **Golden test** from the calibration data. Feed `page.html` through a fake
  browser and `tr_page.json` through a fake linker, as `fake_test.go` and
  `entities_test.go` do. Record these once, through ChromiumFish, into
  `internal/research/testdata/`:
  - the Wikidata `wbgetentities` responses for every QID the run touches;
  - the Wikipedia responses for the keyword lookups and the "Moving"
    disambiguation links;
  - the cosines, from the real model, for the keyword, the page and each
    theme. The fake embedder returns those, so the suite needs neither the
    210 MB model nor live calls.

  Expected results:
  - `about`, in order: Calculator, Moving company, Cost, Relocation
    (personal), Relocation service, Transport, Freight transport, Business
    economics, Pricing, Land transport, Accounting.
  - `mentions`, in order: Truck, Stairs, ZIP Code, Federal Motor Carrier
    Safety Administration, Weight, Piano, Hot tub (`scores`), Billiard table
    (`link`).
  - Email must be dropped as `below_topic` or `below_relevance`, and
    Computing as `below_similarity`.
  - The identity check passes for all 19 calibration QIDs: each Wikidata
    enwiki sitelink title (saved in `relatedness.json`) equals TextRazor's
    `entityId`, and each TextRazor `freebaseId` equals the item's P646
    (`kgmid_check.json`). So every page entity has `kgmid_checked: true`;
    themes and lookups have `false`. If a recorded fixture breaks this, fix
    the compare rule or the fixture, not the expected list.
- **Unit tests:**
  - an identity mismatch, both by title and by Google ID;
  - Bing:
    - a `bingId` attached only when Azure's Wikipedia `url` equals the
      entity's title;
    - a P9885 disagreement leaves the Bing ID out;
    - text over 5,120 characters is split into documents;
    - no Azure settings means no Bing IDs and a note;
    - the Azure key never appears in an error.
  - Use canned Azure responses in the shape of Microsoft's documented
    sample; there is no live Azure data yet;
  - `sameAs` holds exactly Wikipedia, Wikidata, the Google links and
    DBpedia (English pages only);
  - `KGSearch` builds the plain `kgmid` link;
  - `kgmid_checked: false` when TextRazor or Wikidata lacks the ID;
  - a disambiguation item;
  - no borrowed Google ID;
  - `alternateName` leaves out the name;
  - `additionalType` stops at 2;
  - keyword lookups:
    - a disambiguation tie puts every tied option in;
    - a disambiguation with no links puts nothing in;
    - a redirect goes in only when link-confirmed;
    - a missing title is skipped;
  - themes:
    - a category theme is dropped;
    - the name is the sitelink title, not the label;
    - each similarity cutoff on its own drops a theme;
    - no model means no themes and a note;
  - the link path with a link to another host (must not count);
  - the focus note;
  - no TextRazor key.
- No live calls in the test suite. The calibration data holds no API key.

## Left to the builder

- File layout (suggested: `internal/research/selection.go`), type names and
  reason-code spelling.
- How the main-content links are read: an extra field in `pageJS` or a
  second eval.
- Recording the Wikidata, Wikipedia and cosine fixtures.
- The Go binding for the embedding model and its cache location.

## My calls (not the owner's)

- The defaults 0.3, 0.5, 0.40 and 0.55.
- Keyword page entities must also pass the score gate; link-only entities
  go to `mentions`.
- `about` order: keyword page entities by relevance, then lookups in keyword
  order, then themes by topic score. `mentions` by relevance, then topic
  score, then mentions, then name.
- A disambiguation tie keeps every tied option (Relocation (personal) and
  Relocation service).
- A theme is embedded by its label, the way the cutoffs were set.
- Accounting stays in `about`, because it passes both cutoffs. No list is
  edited by hand.
- `name` is the Wikipedia title (as `resolve_entities` uses), not the
  Wikidata label: "Cost", not "monetary cost".
- `additionalType` fills from P279 when P31 gives fewer than 2.
- An identity mismatch drops the entity rather than trusting either side.
- No synonym borrowing for the Google ID.
- A Google ID that differs between TextRazor and Wikidata drops the
  entity; a `freebaseId` only TextRazor has is never used.
- The DuckDuckGo link is not in `sameAs`, reading the owner's five
  families as the full list.
- DBpedia is linked for English pages only, as `schema.Entity` does today.
- The Bing ID goes in `identifier` only, because no Bing link works.
- A Bing ID that differs from Wikidata's P9885 is left out, but the entity
  stays.
- The Google link changes everywhere to the plain `kgmid` form, not only in
  `select_entities`.
- URL input only, no raw-text input.
