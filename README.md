# go-json-schema

Go structs for building schema.org JSON-LD, built for maximal entity
markup: every schema.org type, every property, and helpers for the
identity-heavy patterns (`sameAs` walls, `about`/`mentions` entity lists,
Knowledge Graph IDs).

- **923 types, 1,540 properties, 544 enumeration members.** These come from
  the official vocabulary, including pending, health-lifesci, auto, bib and
  superseded terms.
- **Any value in any property:** text, URL, number, date, nested node,
  `@id` reference, enum member or raw JSON.
- **Multi-type nodes**, `@graph` documents, and `Extra` for properties a
  type doesn't list.

```go
import "github.com/dazamate/go-json-schema/schema"
```

## Composition

Each type is assembled from reusable blocks:

| Block | Holds | Example |
|---|---|---|
| `Node` | `@type`, `@id`, `Extra` | in every type |
| `<Class>Props` | properties only that class defines | `ThingProps` has `name`, `sameAs`, `alternateName`, `additionalType`, `identifier`, `disambiguatingDescription`, ... |
| `<Prop>Trait` | one property several types share | `KnowsAboutTrait` (Person, Organization), `AuthorTrait` (CreativeWork, Rating), `KeywordsTrait` |

`Recipe` embeds `Node`, `ThingProps`, `CreativeWorkProps`, `HowToProps`,
`RecipeProps` and the 50+ traits it can use (`AuthorTrait`,
`AboutTrait`, `AggregateRatingTrait`, `SuitableForDietTrait`, ...). Every
property sits in exactly one block, so embedded fields never clash. Fields
are promoted, so you set them directly:

```go
author := &schema.Person{}
author.ID = "https://hangry.example/#dale"
author.Name = schema.Texts("Dale Woods")                // ThingProps
author.SameAs = schema.URLs("https://www.linkedin.com/in/example") // ThingProps
author.KnowsAbout = schema.Entities(                    // KnowsAboutTrait
	schema.Entity{Name: "Roasting", Wikipedia: "Roasting"},
)

r := &schema.Recipe{}
r.Name = schema.Texts("Lemon herb chicken")
r.Author = schema.Vals(author.Ref())                    // AuthorTrait, Person by @id
r.PrepTime = schema.Vals(schema.DurationOf(15 * time.Minute))
r.SuitableForDiet = schema.Vals(schema.GlutenFreeDiet)  // enum member
r.About = schema.Entities(schema.Entity{Name: "Lemon", Wikipedia: "Lemon"})

tag, _ := schema.ScriptTag(schema.NewGraph(r, author))
```

Generated types (`Person`, `Organization`, `Thing`, ...) are **values**. Put
them in fields (`author`, `founder`, `about`) or link to them with `.Ref()`.
Don't embed them: their `MarshalJSON` would take over your struct.

### Your own composed types

Embed the blocks you need and call `EncodeNode`. This is the route to
multi-type nodes:

```go
type RecipeKit struct {
	schema.Node
	schema.ThingProps
	schema.RecipeProps
	schema.ProductProps
	schema.KeywordsTrait
	schema.SkuTrait
}

func (v RecipeKit) MarshalJSON() ([]byte, error) {
	type plain RecipeKit
	v.Type = v.Type.OrDefault("Recipe")
	return schema.EncodeNode(plain(v), v.Extra)
}

kit := RecipeKit{}
kit.Type = schema.Types{"Recipe", "Product"}
```

A property shared by several types is a trait, not part of a Props
block: `sku` is in `SkuTrait`, because Offer and Demand have it too.
The compiler tells you when a field is missing.

## Values

| Build | Encodes as |
|---|---|
| `schema.Texts("a", "b")` | `["a","b"]` (one value encodes as a scalar) |
| `schema.URLs(...)`, `schema.Vals(schema.URL(...))` | URL strings |
| `schema.Vals(node, schema.Text("x"), ref)` | mixed array |
| `schema.Ref{Type: schema.Types{"State"}, ID: "https://en.wikipedia.org/wiki/Arizona"}` | `{"@type":"State","@id":"..."}` |
| `person.Ref()` | typed reference to a node by its `@id` |
| `schema.Number`, `schema.Integer`, `schema.Boolean` | JSON number / bool |
| `schema.DateOf(t)`, `schema.DateTimeOf(t)`, `schema.DurationOf(d)` | ISO 8601 |
| `schema.VeganDiet` (any enum member) | `"https://schema.org/VeganDiet"` |
| `schema.Raw(...)` | raw JSON, e.g. `{"@list":[...]}` |

Each field's doc comment lists the types schema.org expects. Open fields
don't enforce them, so text-for-node, URL-for-PropertyValue and similar
loose values all encode.

### Enum-only fields

The 75 properties whose only expected type is an enumeration get a
typed list instead of `Values`. The compiler then accepts only:

- that enumeration's members, and members of its sub-enumerations
- a node of that enumeration type
- a `Ref`, for outside IRIs such as GoodRelations

```go
offer.Availability = schema.ItemAvailabilityList{schema.InStock}
offer.ItemCondition = schema.OfferItemConditionList{schema.NewCondition}
hours.DayOfWeek = schema.DayOfWeekList{schema.Monday, schema.Friday}
page.Specialty = schema.SpecialtyList{schema.Cardiovascular} // MedicalSpecialty member, a Specialty sub-enumeration
offer.BusinessFunction = schema.BusinessFunctionList{schema.Ref{ID: "http://purl.org/goodrelations/v1#Sell"}}
```

Every member constant has a typed member type (`InStock` is an
`ItemAvailabilityMember`). Members still work in open `Values` fields:
`suitableForDiet` takes `Diet` nodes too, so it stays open, and
`schema.Vals(schema.VeganDiet, schema.Text("Keto"))` works there.

To put an off-range value in an enum-only property, use `Extra`. It merges
with the typed field:
`offer.Extra = map[string]schema.Values{"availability": schema.Texts("Ships in 24 hours")}`.

## Entity stuffing helpers

```go
seo := schema.Entity{
	Name:      "Search engine optimization",
	Wikipedia: "Search engine optimization",
	Wikidata:  "Q180711",
	KGMID:     "/m/019qb_",
	DBpedia:   "Search_engine_optimization",
	SameAs:    []string{"https://example.com/seo"},
}

seo.Thing()     // one Thing: name + every sameAs URL + GoogleMachineID identifier
seo.PerSource() // one Thing per sameAs URL (the same entity listed once per source)
schema.Entities(a, b, c) // Values of Things, for about / mentions / knowsAbout

schema.Entity{Type: "State", ID: string(schema.Wikipedia("Arizona")), Name: "Arizona"} // areaServed by Wikipedia @id
schema.GoogleMachineID("/m/0f1w9")  // {"@type":"PropertyValue","propertyID":"GoogleMachineID","value":"kg:/m/0f1w9"}
schema.KGSearch("/m/019qb_") // https://www.google.com/search?kgmid=/m/019qb_
schema.Wikipedia, schema.WikipediaLang, schema.Wikidata, schema.DBpedia, schema.ProductOntology
```

## Extra

`Node.Extra` adds properties a struct has no field for:

- off-domain properties (`addressRegion` on a City)
- Google's `query-input` on a SearchAction
- `interactionCount`, the one schema.org property with no domain left

An `Extra` key the struct also sets is merged into one array, never
emitted twice.

```go
search := &schema.SearchAction{}
search.Target = schema.URLs("https://example.com/?s={search_term_string}")
search.Extra = map[string]schema.Values{"query-input": schema.Texts("required name=search_term_string")}
```

## Output

- `schema.JSONLD(v)` adds `@context`. A `Graph` gets `{"@context", "@graph": [...]}`.
- `schema.Marshal(v)` gives compact JSON without HTML escaping, so `&` in
  URLs stays readable.
- `schema.ScriptTag(v)` gives a `<script type="application/ld+json">`
  element, with `<` and `>` escaped so values can't close the tag.

## MCP server: schema-mcp

`cmd/schema-mcp` is an MCP server that researches a keyword the way Zizta's
Entity Explorer does, then checks the JSON-LD an AI writes from it. The
`schema` package doesn't import it, so programs that use only `schema`
don't compile the MCP SDK.

```sh
go install ./cmd/schema-mcp                    # from a checkout; installs to $(go env GOPATH)/bin
claude mcp add schema -- "$(go env GOPATH)/bin/schema-mcp"   # stdio, for Claude Code
schema-mcp -http 127.0.0.1:8765                # streamable HTTP, e.g. for n8n
```

Without a checkout, the repo is private, so set
`GOPRIVATE=github.com/dazamate/*` and run
`go install github.com/dazamate/go-json-schema/cmd/schema-mcp@latest`.

| Tool | Needs ChromiumFish | Returns |
|---|---|---|
| `serp(keyword, country, language, …)` | yes | Top 10 Google results, People also ask, related searches, autocomplete, and `bold_terms`: the words Google bolds in snippets as matches for the keyword. Falls back to DuckDuckGo, then Bing, on a bot wall. |
| `page_terms(links)` | yes | Words and 2–3 word phrases ranked across the ranking pages, each with its rank correlation `rho`; with a TextRazor key, the pages' `entities` and `topics` linked to Wikipedia in context; each page's title, headings and JSON-LD summary; competitors' about/mentions/knowsAbout/keywords merged. |
| `resolve_entities(phrases)` | yes | Wikipedia title, Wikidata QID, aliases, types, `/m/` (P646) and `/g/` (P2671) IDs, and a ready `Thing` node. Status: `exact`, `redirect`, `section`, `ambiguous` or `missing`, with candidates for the last two. |
| `select_entities(url, keyword)` | yes | The `about` and `mentions` entities for your own page and its keyword, each with a ready `Thing` node. Needs a TextRazor key; Bing IDs and themes are optional. See [Entity selection](#entity-selection-select_entities). |
| `type_properties(type)` | no | Every property a type takes, inherited ones included. |
| `enum_values(enum)` | no | Members, sub-enumerations included. |
| `property_info(property)` | no | Domains, expected types, Go field and block. |
| `build_jsonld(draft)` | no | Errors for unknown types, properties and enum members; info for off-domain properties, which are kept. The draft comes back as a script tag. |

`serp` takes these Google search params. Only `keyword` is required.

| Param | Google | Default |
|---|---|---|
| `country` | `gl`, the country searched from | `us` |
| `language` | `hl`, the interface language, e.g. `en`, `pt-BR` | `en` |
| `google_domain` | the domain searched, e.g. `google.com.au` | `google.com` |
| `page` | `start`; ranks continue, so page 2 starts at 11 | `1` |
| `time_range` | `tbs=qdr:`: `hour`, `day`, `week`, `month` or `year` | any time |
| `safe_search` | `safe=active` | off |
| `results_language` | `lr`: only pages in this language | any |
| `results_country` | `cr`: only pages from this country | any |
| `no_autocorrect` | `nfpr=1`: no spelling correction | off |
| `show_omitted` | `filter=0`: include results hidden as very similar | off |
| `web_only` | `udm=14`: the Web filter, links only | off |
| `near` | `near=`: localize to a place, e.g. `Austin, TX` (Cora's "near" button) | none |
| `params` | any other URL params, set last; `q` is reserved | none |

Every search also sends `pws=0` (no personalization) and `num=10`.

A bad value is an error before any request. The DuckDuckGo and Bing
fallbacks use only country and language, and `note` names the params they
ignored.

Location, tested live in October 2026:
- `near` works: `plumber` with `near=Austin, TX` put Austin plumbers at ranks 2 and 3.
- `uule` is ignored in both its name and its lat/long forms.
- Without `near`, results follow the IP's area when `country` matches the
  IP's country. `gl=us` from an Australian IP gave US-wide results.
- `chromiumfish mcp` can't set the browser's geolocation.

Cora SEO Software's other stability settings no longer do anything:
- `google.com/ncr` (no country redirect) now just lands on the google.com
  home page, and google.com didn't redirect anyway.
- `num=100` returns 10 results; Google dropped it in 2025.
- `google.com.au` and other country domains still load without
  redirecting, so `google_domain` stays.

**Google cooldown:** a search, or a Google redirect link followed by
`page_terms` or `resolve_urls`, waits until 30 s have passed since the
previous one ended. Cached results skip it. Each search's autocomplete
lookup runs from the results page as soon as it loads, so it doesn't wait.
- All schema-mcp processes share the cooldown through a locked file,
  `~/.cache/go-json-schema/google-cooldown`.
- A redirect link costs a Google request. So `page_terms` reads other
  sites first, and the agent is told to pass each result's `url` when
  `serp` has one. Ten redirect links take about 5 minutes.
- While a call waits, it sends MCP progress notifications if the client
  asked for them. They keep a client's idle timeout from ending the call.

Prompts: `entity_schema(keyword, page_url, page_type, country, language)` runs the whole workflow, and
`stuffing_guide` gives the volume targets. The guide is copied from
`SEO_FIELD_GUIDE` in recipe-create-pipeline, so update both together, except
the about, mentions, alternateName and additionalType rules, which are
schema-mcp's own.

**Fetching** goes only through [ChromiumFish](https://github.com/arman-bd/chromiumfish).
The server runs `chromiumfish mcp` as a child process, on first use, on a
free CDP port. It finds the command through `CHROMIUMFISH_CMD`, else
`chromiumfish` on PATH, and errors with install steps if either the command
or the downloaded browser is missing. Install it with
`pip install "chromiumfish[mcp]" && chromiumfish fetch`, or with
chromium-fish-mcp's `install.sh`. Wikipedia and Wikidata calls run as
`fetch()` inside a Wikipedia tab: from this machine, direct API requests
are refused with a 403.

**How entities resolve:**
- A phrase is looked up as a Wikipedia title, following redirects. Looking
  up by title instead of searching Wikidata labels keeps "rosemary" from
  matching people named Rosemary.
- The Google IDs come from the matching Wikidata item.
- When that item has no Google ID, the server borrows one from a synonym
  item (P694, P566, P1403, P1420, P460). For example, rosemary's 2020
  taxon item Q102240169 takes `/m/06cd7` from Q122679.

**Settings:**
- `-seed` / `SCHEMA_MCP_PERSONA_SEED`: the browser persona (default `go-json-schema`).
- `-cache` / `SCHEMA_MCP_CACHE_DIR`: the cache directory (default `~/.cache/go-json-schema`).
  Results expire after 1 day for searches, 7 days for pages and 30 days
  for entities. Pass `-cache ''` to turn caching off.
- `-google-cooldown` / `SCHEMA_MCP_GOOGLE_COOLDOWN`: the least time between
  Google requests (default `30s`). `0` turns it off.
- `-config` / `SCHEMA_MCP_CONFIG`: the YAML file of API keys (default
  `~/.config/go-json-schema/schema-mcp.yml`). Copy
  [`schema-mcp.yml.dist`](cmd/schema-mcp/schema-mcp.yml.dist) there,
  then run `chmod 600` on it. The server warns when other users can read
  it, and a misspelt key name is a startup error. A missing default file
  just means no keys.

**Entity linking (TextRazor):** with `textrazor.api_key` set, `page_terms`
sends each loaded page's title and text to [TextRazor](https://www.textrazor.com/docs/rest)
and merges what comes back:
- `entities`: each mention linked to its Wikipedia article and Wikidata
  item using the page's context, so `name` is the exact Wikipedia title to
  pass to `resolve_entities`.
- `topics`: the pages' themes. These are about candidates.
- Sending the text is the one call that bypasses ChromiumFish. Only text
  ChromiumFish already fetched is sent; the `url` parameter, which would
  make TextRazor fetch the page itself, is never used.
- Each page sends at most `max_kb_per_page` (default 30 KB).
- TextRazor bills one request per 10 KB, and the free plan allows 500 a
  day, so 10 pages cost about 30 requests. `entity_note` reports how many
  were used.
- Results are cached for 7 days by the text sent.

**Rank correlation:** with 5 or more pages loaded, every term and entity
gets `rho`, Spearman's correlation between its use on each page and the
page's rank. This is how Cora SEO Software correlates its factors.
- Negative means higher-ranked pages use it more.
- `significant` marks |rho| ≥ 1.96/√(n−1), the 95% two-tailed test.

`-http` has no authentication, and the tools drive a browser whose
sandbox is off (ChromiumFish runs `--no-sandbox`), so bind it to
localhost.

`SCHEMA_MCP_LIVE=1 go test ./internal/research -run TestLive -v` runs the
research tools against the live web.

## Entity selection (`select_entities`)

`select_entities(url, keyword, language?, min_relevance?, min_topic?, min_keyword_similarity?, min_page_similarity?, min_link_confidence?, min_graph_similarity?, min_graph_links?, min_graph_links_ungated?, min_mention_graph_links?, max_graph_neighbours?)`
reads your own page, `localhost` included, and chooses its `about` and
`mentions`. The rules in brief (the full design is
[the spec](docs/superpowers/specs/2026-10-25-own-page-entities-design.md)):
- Every entity comes from your page's text, linked to Wikipedia and Wikidata
  by TextRazor in context, then checked by its Wikidata item. Nothing is
  searched by name. The one name-based step is the keyword's words and
  phrases looked up as exact Wikipedia titles.
- `about` is the keyword's own entities (named on the page, or an exact
  title) followed by the page's themes, which the page need not name.
- `mentions` holds only entities the page names.
- When the keyword has its own Wikipedia article (the whole keyword, or a
  phrase of 2 or more of its words), `about` is instead that article, the
  articles linked to or from it that are close to the keyword or named on
  the page (at most `max_graph_neighbours` unnamed ones), and the page
  entities well linked into that graph. `mentions` then holds only page
  entities linked into it, and `graph` reports its size. With no article,
  or fewer than 3 kept neighbours, `about` falls back to the page's themes
  and `note` says so. The design is
  [the keyword graph](docs/superpowers/specs/2026-10-25-keyword-graph-design.md).
- Neither list has a cap beyond that one. `dropped` lists what was rejected
  and why (`off_graph`, `below_graph_similarity`).
- Each entry has a `thing` node to use as given.

**Defaults:** `min_relevance` 0.3, `min_topic` 0.5, `min_keyword_similarity`
0.40, `min_page_similarity` 0.55 and `min_link_confidence` 1.5. The first
four are 0 to 1. `min_link_confidence` is a TextRazor confidence (about 0.5
to 10) that an entity kept only by a link to another page of your site must
reach. The graph settings are `min_graph_similarity` 0.40 (0 to 1, for an
article the page does not name), `min_graph_links` 3, `min_graph_links_ungated`
10 and `min_mention_graph_links` 2 (whole numbers of 1 or more) and
`max_graph_neighbours` 30 (0 or more; the owner set this cap). These are our judgement, set by eye
from one calibration page (the Lift My Boxes moving cost calculator page,
keyword "moving cost calculator", 2026-10-25), not from any Google source.
They still need checking on a second page. The calibration data is in
[`docs/superpowers/specs/2026-10-25-own-page-entities/`](docs/superpowers/specs/2026-10-25-own-page-entities/).

**IDs it emits:** the Wikipedia URL, Wikidata QID, Google Knowledge Graph
IDs (`/m/` from P646, `/g/` from P2671, linked as
`https://www.google.com/search?kgmid=<id>`), DBpedia (English pages only)
and, with Azure, the Bing entity ID. IDs come only from those sources and
are never filled from a synonym item.

**Bing IDs (Azure AI Language):** set `azure_language.endpoint` and
`azure_language.api_key` in the config file (set both or neither; see
`schema-mcp.yml.dist`). Only English and Spanish pages are read. Azure's
entity linking retires on 1 September 2028; after that Bing IDs stop and
the rest keeps working. Without the settings, `note` says no Bing IDs were added.

**Themes (local model):** similarity uses BAAI/bge-base-en-v1.5 (quantized
ONNX) and ONNX Runtime, downloaded on first use (about 220 MB, pinned by
SHA-256) under `~/.cache/go-json-schema/models`. This is linux/amd64 only
and needs cgo and gcc. Without them, or without the download, the tool still
runs with no themes and `note` says why. `CGO_ENABLED=0 go build ./...`
works.

**Tests:** the default `go test ./...` makes no live calls.
- `SCHEMA_MCP_MODEL=1 go test ./internal/embed` runs the real model.
- `SCHEMA_MCP_RECORD=1 go test ./internal/research` re-records the fixtures.
- `SCHEMA_MCP_LIVE=1 go test ./internal/research -run TestLive -v` uses the real browser.

## Regenerating

The vocabulary is stored in the repo at
`internal/schemagen/vocab/schemaorg-all-https.jsonld`. It was downloaded
2026-10-23 from
`https://schema.org/version/latest/schemaorg-all-https.jsonld`, when the
site was on V30.1. To update:

```sh
curl -sL -o internal/schemagen/vocab/schemaorg-all-https.jsonld \
  https://schema.org/version/latest/schemaorg-all-https.jsonld
go generate ./schema
go test ./...
```

`go generate` fails if a generated name clashes with handwritten code.
`TestGeneratedFilesUpToDate` fails if the `*_gen.go` files are stale.
`TestEveryTypeEncodesEveryApplicableProperty` fills every field of every
type, encodes it, and checks the keys against the vocabulary.

### Naming

- Property fields are UpperCamel, with `Url` and `Id` written as `URL` and
  `ID`. For example, `contentUrl` becomes `ContentURL`.
- Types and enum members keep their schema.org spelling. A leading digit
  gets an `X` prefix: `3DModel` becomes `X3DModel`, whose `@type` is still
  `3DModel`.
- Datatypes and quantities (`Text`, `URL`, `Number`, `Duration`, `Energy`,
  `Mass`, ...) are Go value types, not structs.
