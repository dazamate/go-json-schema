# One Check Per File Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split build_jsonld's audit (`internal/vocabtool`'s one `checker` struct) into one registered check per file behind an `audit.Check` interface, each tested alone, with no change to what the audit finds.

**Architecture:** The work happens in three stages.

1. Core packages (`vocab`, `jsonld`, `txt`, `webpage`, `audit`) take the shared code out of `vocabtool`.
2. The old `checker` becomes a temporary legacy adapter check run by `audit.Run`. Area by area, its checks move to `internal/vocabtool/checks/<area>`, one per file, and are deleted from the adapter.
3. When the adapter is empty, it is deleted.

Two ledgers in `audit` replace the shared mutable maps (`claimed` and `suggested`) with facts computed in a fixed, written order.

**Tech Stack:** Go 1.26 module `github.com/dazamate/go-json-schema`, plus `golang.org/x/net/html` and `github.com/andybalholm/cascadia`.

**Spec:** `docs/superpowers/specs/2026-11-05-one-check-per-file-design.md`. Read it first; this plan argues from it.

## Global Constraints

- **Golden gate.** `go test -count=1 -run TestGolden ./internal/vocabtool/` passes after every commit. A step that breaks it has a bug in that step. Never edit `golden_test.go` or `testdata/golden/` before Task 19.
- **Full suite.** `go vet ./... && go test -count=1 ./...` passes at every commit.
- **No output changes.** No message, path, severity or count changes. No new checks. No change to the JSON shape of `Report` or `Issue`.
- **Public API.** The `check` package (`check.Vocabulary`, `check.Errors`) and `internal/mcpserver` compile unchanged. `vocabtool.Issue`, `vocabtool.Report`, `vocabtool.Error`, `vocabtool.Warning`, `vocabtool.Info` and `vocabtool.CheckOptions` keep their names.
- **Existing tests stay.** The existing `_test.go` files in `internal/vocabtool` stay. Edit them only to:
  - follow a renamed unexported identifier;
  - fix an assertion that depends on issue order.
- **Package names.** Core: `vocab`, `jsonld`, `txt`, `webpage`, `audit`, `audit/audittest`. Areas: `checks/terms`, `graph`, `snippets`, `onpage`, `content`, `figures`, `shop`, `product`, `policies`, `costs`, `site`, `sources`, `keyword`.
- **Check files.**
  - One exported struct per file.
  - ID `"<area>/<kebab-name>"`. File `<snake_name>.go`, type `CamelName`.
  - Its test is `<snake_name>_test.go` in package `<area>_test`.
  - It registers in `init()`.
- **Granularity.** One check per distinct `c.add` message template. A template built from a loop over fields, or one that only fills in a word, stays one check.
- **Helpers.** A helper used by one area goes in a helper file of that area, named for what it holds. A helper used by several areas goes to the core package matching what it reads:
  - draft values: `jsonld`;
  - the rendered page: `webpage`;
  - plain strings, URLs, amounts: `txt`;
  - the vocabulary: `vocab`;
  - the context or ledgers: `audit`.
- **Commits.** Commit at the end of each task. Message: conventional (`refactor(vocabtool): …`), ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

These are failure modes no per-task test pins unless the task adds the test named:

1. **Concurrent calls.** The MCP server may call `Tool.Check` concurrently. The registry is written only in `init()`, and the memo and ledgers live on the per-call `Context`, so two calls share nothing mutable. Task 4 adds `TestCheckConcurrent`, run with `-race`.
2. **A gate lost in the move.** A check must keep the exact condition today's call chain puts on it:
   - `content()`: needs page facts, and needs an indexed page for all but five checks;
   - `shop()`: `Doc != nil`, `kind == product`, `Sells()`;
   - `site()`: `Doc != nil`;
   - `figures()`: `Doc != nil`;
   - `costs()`: `!Sells()`;
   - `keyword`: needs a keyword.

   The golden gate only covers recorded pages. Each task's check tests therefore add a "gated off" quiet case for every gate the check has.
3. **The array root in `Vocabulary`.** `Vocabulary` must keep `$[0]` paths: no `@graph` wrapping. Task 4 adds `TestVocabularyArrayRootPaths`.
4. **The script tag after `Normalize`.** It must match today's byte for byte, including the expansion of bare enum names nested in untyped objects and inside `@graph`. Task 4 adds `TestNormalizeMatchesWalk` beside the golden.
5. **Duplicate or misplaced IDs.** Two checks with one ID, or an ID whose area isn't its directory, would silently shadow a check. `audit.Register` panics on a duplicate (Task 4, `TestRegisterDuplicatePanics`), and Task 19's meta-test checks each ID's directory.

## The area move procedure

Tasks 5 to 18 each move a set of legacy `checker` methods to an area package. They all follow this procedure; the task lists only what differs.

1. **Inventory.** For each legacy method in the task, list every `c.add` call. Group the calls by message template; each group becomes one check. Write down its gate: the conditions on the call path from `audit`'s legacy adapter down to the call (see Review Focus 2).
2. **For each check, write its test first** (`<snake_name>_test.go`, package `<area>_test`):
   - A table with at least one firing case that asserts the full `[]audit.Issue`: path, severity and message.
   - At least one quiet case, plus one quiet case per gate.
   - Reuse inputs from the existing tests in `internal/vocabtool/*_test.go` where they hit this check.
   - Use `audittest.Expect`.
3. **Run the test.** It fails to compile (no such type).
4. **Write the check file.** Move the method's logic into `Run`, reading from `*audit.Context`, and give it its gate. Regexes and tables only it uses move with it. Shared helpers go to the area's helper file, or to core per Global Constraints.
5. **Run the test again.** It passes.
6. **Delete the moved code** from the legacy adapter's call chain and from the legacy files. Delete any legacy file that is now empty. A legacy helper still used by legacy code stays until its last user moves.
7. **Run the gates.** The golden gate passes, then `go vet ./... && go test -count=1 ./...` passes.
8. **Commit.**

---

### Task 1: The `vocab` package

**Files:**
- Create: `internal/vocabtool/vocab/vocab.go` and `internal/vocabtool/vocab/subtypes.go`, holding today's `Tool` index fields and `subtypes.go`'s non-checker code.
- Create: `internal/vocabtool/vocab/vocab_test.go`.
- Modify: `internal/vocabtool/vocabtool.go`, `internal/vocabtool/subtypes.go`, `internal/vocabtool/check.go`, and every legacy file that reads `c.t.v`, `c.t.applicable`, `c.t.enumOnly`, `c.t.suggest`, `c.t.members`, `c.t.descendants`, `c.t.subtypesTaking`, `c.t.mostGeneral` or `c.t.suggestable`.

**Interfaces:**
- Produces:
  ```go
  package vocab
  type Vocab struct { V *schemagen.Vocab /* plus the unexported indexes */ }
  func Load() (*Vocab, error)                          // today's New() body
  const KindType, KindProperty, KindMember = "type", "property", "member"
  func (v *Vocab) Applicable(ty, prop string) bool
  func (v *Vocab) EnumOnly(prop string) (enum string, ok bool)
  func (v *Vocab) Members(enum string) []string
  func (v *Vocab) Suggest(kind, name string) string
  func (v *Vocab) Suggestable(name string) bool
  func (v *Vocab) Descendants(class string) []string
  func (v *Vocab) MostGeneral(cands []string) []string
  func (v *Vocab) SubtypesTaking(types []string, prop string) []string
  func (v *Vocab) RangesEnum(prop string) bool
  func (v *Vocab) Is(types []string, ty string) bool  // some type is ty or a subtype of it
  ```
- `vocabtool.Tool` becomes `struct{ *vocab.Vocab }`. `New()` calls `vocab.Load()`.
- The `checker` field `t *Tool` becomes `v *vocab.Vocab`.

- [ ] **Step 1: Write `vocab_test.go`.**
  - `TestSuggest`: `Suggest(KindType, "Prodcut") == "Product"`.
  - `TestSubtypesTaking`: `SubtypesTaking([]string{"Organization"}, "openingHours")` contains `"LocalBusiness"`.
  - `TestIs`: `Is([]string{"Vehicle"}, "Product")` is true; `Is([]string{"Place"}, "Product")` is false.
  - `TestEnumOnly`: `EnumOnly("availability")` returns `"ItemAvailability", true`.
- [ ] **Step 2: Run the test.** `go test ./internal/vocabtool/vocab/` fails: no package.
- [ ] **Step 3: Move the code.** Move the index and its methods into `vocab`, and update `vocabtool` to use it. Existing tests that call the old unexported names switch to the new exported ones.
- [ ] **Step 4: Run the test and the global gates.** The vocab test, the golden gate and the full suite pass.
- [ ] **Step 5: Commit.** `refactor(vocabtool): vocabulary index in its own package`

### Task 2: The `jsonld` and `txt` packages

**Files:**
- Create: `internal/vocabtool/jsonld/`. Holds:
  - the ordered object from `check.go`: `object`, `decodeOrdered`, `decodeValue`, `encode`;
  - `cutSchemaURL` and `stripPrefix`;
  - `wikidataItem` and `wikidataItems`;
  - `pagecheck.go`'s `located`, `nodes`, `typeNames`, `values` and `strs`;
  - `content.go`'s `scalars` and `draftIndex`;
  - `product.go`'s `hasKey`, `keyMember`, `deepScalars` and `under`;
  - `site.go`'s `byID`, `refs` and `publishers`;
  - `snippets.go`'s `described`;
  - `subtypes.go`'s `ownItems` and `hasWikidataClass`.

  Each goes only if it reads draft values alone. A helper that needs the vocabulary stays for Task 4's `Context`.
- Create: `internal/vocabtool/txt/`. Holds:
  - `normWords` and `names` (whole-word matching);
  - `clip`, `stripTags` and `examples`;
  - `normURL` and `normLink`;
  - `amount`, `priceOf` and the `priceAmount` regex;
  - `digits` and `digitsOf`.
- Tests: `jsonld/object_test.go`, `jsonld/nodes_test.go`, `txt/words_test.go`, `txt/amount_test.go`.
- Modify: every legacy file, to use the moved names.

**Interfaces:**
- Produces (the builder picks the remaining names):
  ```go
  package jsonld
  type Object struct { Keys []string; Vals map[string]any }
  func (o *Object) Get(k string) (any, bool)
  func (o *Object) Set(k string, v any)
  func (o *Object) Prepend(k string, v any)
  func Decode(data []byte) (any, error)          // keeps key order, json.Number
  func Encode(v any, pretty bool) ([]byte, error) // no HTML escaping
  type Node struct { Path string; N *Object }
  func Nodes(path string, v any) []Node           // every object, outermost first
  func TypeNames(n *Object) []string
  func Values(raw any) []any
  func Strs(n *Object, key string) []string
  func Scalars(n *Object, key string) []string
  type Index struct { Strs []string; URLs map[string]bool; Path map[*Object]string }
  func NewIndex(all []Node) *Index

  package txt
  func NormWords(s string) string
  func Names(paddedNormPage, form string) bool
  func Clip(s string, n int) string
  func NormURL(u *url.URL) string
  func NormLink(s string) string
  func Amount(s string) (float64, bool)
  ```

- [ ] **Step 1: Write the tests.**
  - `TestDecodeKeepsOrder`: `Decode({"b":1,"a":2})` then `Encode` gives `{"b":1,"a":2}`.
  - `TestEncodeNoHTMLEscape`: `"<b>"` encodes as `"<b>"`.
  - `TestNodesPaths`: `{"@graph":[{"x":{"y":{}}}]}` gives the paths `$`, `$.@graph[0]`, `$.@graph[0].x`, `$.@graph[0].x.y`.
  - `TestNormWords`: `"Pre-wedding"` gives `"pre wedding"`.
  - `TestNames`: `Names(" brides ", "Bride")` is true.
  - `TestAmount`: `Amount("1,250.50")` gives `1250.5`.
- [ ] **Step 2: Run the tests.** They fail: no package.
- [ ] **Step 3: Move the code and update the legacy files.**
- [ ] **Step 4: Run the tests and the global gates.** All pass.
- [ ] **Step 5: Commit.** `refactor(vocabtool): draft model and text helpers in their own packages`

### Task 3: The `webpage` package

**Files:**
- Create: `internal/vocabtool/webpage/`. Holds:
  - `pageFacts` as `Facts`, and `newPageFacts` as `New`;
  - `mainText`, `mainLines`, `canonical`, `queryAll`, `shown` and `abs`;
  - the page kinds and `classify`;
  - `shop.go`'s page readers: `segments`, `h1`, `metaContent`, `noindex`, `buyForm`, `productCards` and `card`, `siteLink(s)`, `holds`, `relatedBoxes`, `sells`, `promo`, `listingLink`, `homeLink`, `businessName`, `storeType` and `address`;
  - `policies.go`'s `questions`;
  - `data.go`'s `dataBlocks`;
  - `site.go`'s `tools` and `phases`;
  - `product.go`'s `optionGroups`, `productDetails`, `wasPrices` and `title`;
  - `elements.go`'s HTML helpers and `pageSections`;
  - `content.go`'s HTML helpers used by several areas.

  Move only what more than one area reads. The builder judges this from the area table in the spec, and the rest stays for its area task. `claimed` and `claim` stay on the legacy side until Task 5.
- Tests: `webpage/facts_test.go`.
- Modify: every legacy file that reads `pageFacts`.

**Interfaces:**
- Produces:
  ```go
  package webpage
  type Kind string                       // today's pageKind; constants keep their names, exported (KindProduct, KindListing, …)
  type Facts struct {
      Doc  *html.Node; Text, Norm, Main string; Base, URL *url.URL
      Kind Kind; Private string
      /* unexported memo for cards */
  }
  func New(doc *html.Node, text, address string) *Facts
  func (p *Facts) Sells() bool
  func (p *Facts) ProductCards() []*Card
  func HTMLText(n *html.Node) string
  func SelectorFor(doc, n *html.Node) string
  func QueryAll(n *html.Node, sel string) []*html.Node
  ```

- [ ] **Step 1: Write `facts_test.go`.**
  - `TestNewClassifiesProductPage`: a page with an add-to-cart form and an `og:type` product meta gives `Kind == KindProduct`. Take the input from `shop_test.go`'s product case.
  - `TestNewPrivateNoindex`: `<meta name="robots" content="noindex">` gives `Private == "noindex"`.
  - `TestMainSkipsChrome`: the text of a site `<header>` is not in `Main`.
- [ ] **Step 2: Run the test.** It fails: no package.
- [ ] **Step 3: Move the code and update the legacy files.**
- [ ] **Step 4: Run the tests and the global gates.** All pass.
- [ ] **Step 5: Commit.** `refactor(vocabtool): page facts in their own package`

### Task 4: The `audit` framework, the harness and the legacy adapters

**Files:**
- Create: `internal/vocabtool/audit/check.go` (interface, `Issue`, severities, registry); `audit/context.go`; `audit/normalize.go`; `audit/run.go`; `audit/ledger.go`; `audit/audittest/audittest.go`.
- Tests: `audit/run_test.go`, `audit/normalize_test.go`, `audit/ledger_test.go`.
- Create: `internal/vocabtool/legacy.go`, with the two adapter checks wrapping the old `checker`.
- Modify: `internal/vocabtool/check.go`. `Check` and `Vocabulary` build an `audit.Context` and call `audit.Run`. `Issue` becomes `= audit.Issue`, and the severity constants become `= audit.Error` and so on.
- Tests: `internal/vocabtool/check_test.go` (new), with `TestCheckConcurrent`, `TestVocabularyArrayRootPaths` and `TestNormalizeMatchesWalk`.

**Interfaces:**
- Consumes: Tasks 1 to 3.
- Produces:
  ```go
  package audit
  const ( Error = "error"; Warning = "warning"; Info = "info" )
  type Issue struct {
      Path     string `json:"path"`
      Severity string `json:"severity"`
      Message  string `json:"message"`
  }
  func Issuef(path, severity, format string, args ...any) Issue
  type Check interface { ID() string; Run(ctx *Context) []Issue }
  func Register(c Check)                 // panics on an empty or duplicate ID
  func Checks() []Check                  // in registration order
  func Group(prefix string) []Check      // the checks whose ID starts with prefix
  type Input struct { PageText, PageHTML, PageURL, Keyword string; KeywordDensity float64 }
  type Context struct {
      Vocab *vocab.Vocab
      Root  any              // normalized draft root: *jsonld.Object, or []any for Vocabulary
      Nodes []jsonld.Node    // every object, outermost first
      HTML  *html.Node       // parsed page_html, nil without it
      Page  *webpage.Facts   // nil without page text or html
      Known *webpage.Facts   // Page, or facts holding only the page URL
      In    Input
  }
  func NewContext(v *vocab.Vocab, draft []byte, in Input) (*Context, error)    // decode, wrap an array in @graph, prepend @context, Normalize, parse html, facts; today's errors verbatim
  func NewVocabularyContext(v *vocab.Vocab, draft []byte) (*Context, error)    // decode, Normalize, no wrapping
  func Normalize(v *vocab.Vocab, root any)
  func (c *Context) Is(n *jsonld.Object, ty string) bool
  func (c *Context) Single(n *jsonld.Object) string   // today's checker.single
  func (c *Context) Has(ty string) bool
  func (c *Context) Index() *jsonld.Index             // memoized
  func (c *Context) Indexed() bool                    // Page != nil && Page.Private == ""
  func Memo[T any](c *Context, key any, f func() T) T
  func Run(c *Context, checks []Check) []Issue        // stable sort: severity (error, warning, info), then check order, then emission

  // price ledger
  type PriceTaker interface { Check; TakesPrices(c *Context, before Taken) Taken }
  type Taken map[float64]bool
  func (c *Context) TakenBefore(id string) Taken      // union of what the takers before id take, in priceOrder
  var priceOrder []string                             // filled by Task 5

  // retype ledger
  type Retyper interface { Check; Retypes(c *Context, before Retyped) []Retype }
  type Retype struct { Node *jsonld.Object; Named []string }
  type Retyped struct { Told map[*jsonld.Object]bool; Named map[*jsonld.Object][]string }
  func (c *Context) RetypedBefore(id string) Retyped
  var retypeOrder []struct{ ID string; Blocks bool }  // filled by Task 6

  package audittest
  type Input struct { Draft, PageHTML, PageText, PageURL, Keyword string; KeywordDensity float64 }
  func Run(t testing.TB, c audit.Check, in Input) []audit.Issue          // NewContext, then c.Run alone
  func Expect(t testing.TB, c audit.Check, in Input, want []audit.Issue) // nil and empty compare equal; prints both on mismatch
  ```
- **Legacy adapters.** `legacyTerms` (ID `terms/legacy`) runs today's `walk`. `legacyAll` (ID `legacy/all`) runs everything else in today's order.
  - Both are registered in `internal/vocabtool/legacy.go`.
  - Both build a `checker` from the context and return its issues.
  - `legacyAll` stores the page elements with `Memo` under an exported key. `Check` copies them into `Report.PageElements`.
- **`checks/all`** is a package that blank-imports every area. `vocabtool` and `audittest` both blank-import it, so production and tests register the same checks. It is created here with no imports yet; each area task adds its line. `audittest` loads the vocabulary once (`sync.Once`).

- [ ] **Step 1: Write the audit tests.**
  - `TestRunSortsBySeverity`: two fake checks, registered A then B, emit `[info, error]` and `[warning, error]`. `Run` returns A's error, B's error, B's warning, A's info.
  - `TestRegisterDuplicatePanics`.
  - `TestNormalizeExpandsEnum`: `{"@type":"Offer","availability":"InStock"}` becomes `"https://schema.org/InStock"`.
  - `TestNormalizeLeavesUnknown`: `"availability":"Nope"` stays.
  - `TestNormalizeSkipsForeignKeys`: `"ex:availability":"InStock"` stays.
  - `TestNormalizeUntypedNested`: an enum inside an untyped object under a property is expanded.
  - `TestTakenBefore`: three fake takers in a test-only order. The second sees the first's amounts; the third sees both.
  - `TestRetypedBefore`: a non-blocking source's node isn't in `Told` but is in `Named`.
- [ ] **Step 2: Write the vocabtool tests.**
  - `TestCheckConcurrent`: 8 goroutines run `Check` on `testdata/lmb/2-movers-cost.*`, and each report equals a serial run.
  - `TestVocabularyArrayRootPaths`: `[{"@type":"Thng"}]` gives the path `$[0]`.
  - `TestNormalizeMatchesWalk`: for every golden case, `NewContext`'s normalized root encodes to the same bytes as the golden script tag's JSON. Encode it the way `Check` does.
- [ ] **Step 3: Run the tests.** They fail: no package or symbols.
- [ ] **Step 4: Build `audit`, `audittest`, `checks/all` and the adapters, and rewire `Check` and `Vocabulary`.** `Check` assembles the `Report`: the counts from the issues, the page elements from the memo, the script tag from `jsonld.Encode(ctx.Root, opts.Pretty)` with `<` and `>` escaped as today.
- [ ] **Step 5: Run the tests and the global gates.** `go test -race ./internal/vocabtool/...` passes, and so do the golden gate and the full suite.
- [ ] **Step 6: Commit.** `refactor(vocabtool): run the audit through audit.Check, legacy checker as an adapter`

### Task 5: The price ledger and the checks that take prices

**Moves** (area move procedure):
- `shop.go`'s `listing` and `sortOrder` to `checks/shop`;
- `product.go`'s `sale` and `extras` to `checks/product`;
- `policies.go`'s `shipping` to `checks/policies`;
- `data.go`'s `data` to `checks/sources`;
- `costs.go`'s `breakdowns`, `rates`, `goods`, `estimates` and `service`, and `content.go`'s `prices`, to `checks/costs`.

**Ledger:**
- Fill `audit.priceOrder` with the moved checks' IDs in today's order: listing, sale, extras, shipping, data, breakdowns, rates, goods, estimates.
- Each check that claims implements `audit.PriceTaker`. It takes exactly the amounts today's code claims, including claims made when nothing is reported. Its `Run` and `TakesPrices` share one find function in its file.
- `breakdowns`, `rates`, `goods` and `estimates` compute `fresh` from `TakenBefore(own ID)` and the draft's amounts (today's `amounts`, now a costs helper).
- `prices` reads `TakenBefore("costs/prices")`. `"costs/prices"` is last in the order and takes nothing.
- Delete `pageFacts.claimed` and `claim`.

**Gates:**
- listing: indexed page.
- sale, extras, shipping: indexed page and `Sells()`.
- data: indexed page.
- costs checks: indexed page and `!Sells()`. `breakdowns` also needs `Doc != nil`.
- prices: indexed page.

**Extra test:** `TestPricesSkipsTaken` in `checks/costs`. A non-selling page shows "$40 an hour" and no other price. `costs/rates` fires, and `costs/prices` stays quiet on $40.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): price ledger; checks that take prices, one per file`

### Task 6: The retype ledger, and the vocabulary walk as the `terms` area

**Moves:**
- `check.go`'s `walk`, `types`, `unknown`, `property` and `value` to `checks/terms`. The traversal goes in `checks/terms/walk.go` with today's early exits (spec, "The vocabulary walk"). Then delete `legacyTerms`.
- `subtypes.go`'s `classTypes` and `namedTypes` to `checks/graph`.
- `shop.go`'s `pageType` and `business`, all their messages, to `checks/shop`.
- `site.go`'s `onlineBusiness` to `checks/site`.

**Ledger:** fill `audit.retypeOrder`:

| # | ID | Blocks |
|---|---|---|
| 1 | `terms/subtype-property` | true |
| 2 | `graph/class-type` | true |
| 3 | `graph/named-type` | false |
| 4 | `shop/page-kind-type` | true |
| 5 | `shop/business-type` | true |
| 6 | `site/online-business` | true |

- `namedTypes` skips `Told` nodes.
- `pageType` skips a node in `Told`, or one whose `Named` holds the subtype it wants. If the golden differs, match today's substring rule on the first named subtype (spec).
- `business` and `onlineBusiness` skip `Told`.
- Delete `checker.suggested`.

**`Vocabulary`:** runs `audit.Group("terms/")`.

**Gates:**
- `pageType`, `business`, `onlineBusiness`: indexed page and `Doc != nil`.
- `business`: also `Sells() || Address() != ""`.
- `onlineBusiness`: also its own conditions today.
- terms and graph checks: none.

**Extra test:** `TestPageTypeQuietAfterNamedType` in `checks/shop`. A WebPage at `/about/` on a page classified as about gives `graph/named-type`'s warning and no `shop/page-kind-type` warning.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): retype ledger; the vocabulary walk's checks, one per file`

### Task 7: `snippets`

**Moves:** `snippets.go`, all of it, to `checks/snippets`. `resolve`, `byID` and the offers walk go in `checks/snippets/offers.go`. Delete `snippets_test.go`'s order-dependent matching only if it now fails, and make it order-free.

**Gate:** none.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): product snippet checks, one per file`

### Task 8: The rest of `graph`

**Moves:**
- `repeats` and `webPageElements`;
- `bare`, `ownReviews` and `reviews`;
- `offers`, `personNames` and `pageNames`.

**Gates:**
- `offers`, `personNames` and `pageNames`: `Page != nil` (they run inside `content()` before the private return).
- The rest: none.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): draft-only graph checks, one per file`

### Task 9: `onpage`

**Moves:** `pageEntities`, `pageElements`, `shownText` and `anchors`.

**Page elements:** `pageElements` becomes a memoized fact in `checks/onpage/elements.go` (helper file), exported as `onpage.Elements(c *audit.Context) []PageElement`, with its `warn` flag (`Page == nil || Page.Private == ""`). The section checks report from it. `vocabtool.Check` takes `Report.PageElements` from `onpage.Elements`, and `legacyAll` stops storing them. `PageElement` moves to `onpage` and stays aliased in `vocabtool`.

**Gates:**
- `pageEntities`: page text, from `PageText` or the HTML's text, is not blank.
- `pageElements`: `HTML != nil`.
- `shownText` and `anchors`: `Page != nil`.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): page-match checks, one per file`

### Task 10: `keyword`

**Moves:** `keyword`, `organizations` and `looseNodes`, with `keyword.go`'s helpers.

**Gates:**
- `keyword`: `strings.TrimSpace(In.Keyword) != ""`. It reads `Known`.
- `organizations` and `looseNodes`: none. They read `Known`.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): keyword checks, one per file`

### Task 11: The rest of `site`

**Moves:** `publisher`, `tollFree`, `pageDates`, `article`, `checklists` and `tools`.

**Gates:**
- all: indexed page;
- `publisher`: also `Doc != nil`.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): site checks, one per file`

### Task 12: The rest of `sources`

**Moves:** `rules`, `permits`, `agencies`, `documents` and `citations`. Find where `documents` is called from today, and keep that gate.

**Gate:** indexed page.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): source checks, one per file`

### Task 13: `figures`

**Moves:** `figures.go`, all of it: `charts`, `summary`, `vehicles`, `definitions`, `quantities` and `options`.

**Gate:** indexed page and `Doc != nil`.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): figure checks, one per file`

### Task 14: The rest of `policies`

**Moves:** `returns`, `payments`, `whereAndWhen` and `faq`.

**Gates:**
- `returns` and `payments`: indexed page and `Sells()`.
- `whereAndWhen` and `faq`: indexed page.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): policy checks, one per file`

### Task 15: The rest of `product`

**Moves:**
- `product`, `choices`, `sizes` and `details`;
- `stock`, `members`, `engagement` and `founding`.

**Gates:**
- `product`, `choices`, `sizes` and `details`: indexed page and `Kind == KindProduct`. Confirm today's chain: `product` calls `choices`, `sizes` and `details`.
- `stock` and `members`: indexed page and `Sells()`.
- `engagement` and `founding`: indexed page.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): product checks, one per file`

### Task 16: The rest of `shop`

**Moves:** `website`, `logo`, `social` and `catalog`.

**Gate:** indexed page and `Doc != nil`.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): shop checks, one per file`

### Task 17: `content`, part 1

**Moves:**
- `content()`'s private-page note, as `content/private-page`. It is gated on `Page != nil && Page.Private != ""`.
- `copyright`, `readTime`, `sectionLabel`, `media` and `search`;
- `contacts`, `forms`, `registrations` and `terms` (as `content/defined-terms`).

**Gate:** indexed page, except the private note.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): content checks (1 of 2), one per file`

### Task 18: `content`, part 2

**Moves:**
- `images`, `counts`, `outsideLinks` and `related`;
- `responseTime`, `quotes` and `ratings`.

**Gate:** indexed page.

After this task, `legacyAll` should run nothing. Confirm that its `Run` body is empty.

- [ ] **Step 1: Follow the area move procedure.**
- [ ] **Step 2: Commit.** `refactor(vocabtool): content checks (2 of 2), one per file`

### Task 19: Finish

**Files:**
- Delete: `internal/vocabtool/legacy.go`, the `checker` type, and every legacy file left empty.
- Create: `internal/vocabtool/registry_test.go`, with `TestEveryCheckHasATest` and `TestCheckIDsMatchDirectories`.
- Modify: `README.md` and `internal/mcpserver/guide.md` wherever they name vocabtool's files or layout. Add a short "Adding a check" note to the README: one file, its test, its `init()`, and the ledger lists for price or retype checks.
- Delete: `internal/vocabtool/golden_test.go` and `internal/vocabtool/testdata/golden/`. Do this last, only after Steps 1 to 4 pass.

- [ ] **Step 1: Write the registry tests.**
  - `TestEveryCheckHasATest`: for each `audit.Checks()`, the file `checks/<area>/<snake(name)>_test.go` exists. Snake-case the ID's name part.
  - `TestCheckIDsMatchDirectories`: each ID's area is a directory under `checks/`. No `legacy` ID is left.
- [ ] **Step 2: Run them.** They pass, or they name the missing files. Add those files.
- [ ] **Step 3: Delete the adapter and `checker`, and update the docs.**
- [ ] **Step 4: Run the gates.** The golden gate passes one last time, then the full suite passes, then `go test -race ./internal/vocabtool/...` passes.
- [ ] **Step 5: Delete the golden test and its data, and run the full suite.** It passes.
- [ ] **Step 6: Commit.** `refactor(vocabtool): drop the legacy checker; every check in its own file`
