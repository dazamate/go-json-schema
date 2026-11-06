# One check per file: splitting build_jsonld's audit

Date: 2026-11-05
Branch: `refactor/audit-checks`, cut from PR #12 (`feat/product-snippet-checks`) with PR #11 (`feat/public-vocabulary-check`) merged in.

## Why

build_jsonld's audit lives in `internal/vocabtool`. Today one `checker` struct does all of it:

- 107 methods;
- 184 places that add a finding;
- about 7,600 lines across 20 files, with `content.go` alone at 1,700.

`Check()` and `content()` call about 60 checks in a fixed order that nobody can see. Checks talk to each other through shared mutable state:

- `pageFacts.claimed`: prices one check reports, which later checks skip.
- `checker.suggested`: nodes already told to take a subtype.
- `pageType` reads other checks' issue messages.

Tests mostly run the whole audit and match messages with regexes. One check can't be tested alone.

The owner wants the audit built the way they would build it as a software engineer:

- an interface every check implements (strategy pattern);
- each kind of issue its own struct, one check per file;
- each check tested in isolation;
- no god object.

Adding a check for something a report such as Search Console flags should mean writing one new file and its test. Changing one should mean editing that file and its test.

## Goals

- **One check per file.** Every kind of finding is its own exported struct in its own file and implements one interface.
- **Isolated tests.** Each check has a test file beside it that runs that check alone and asserts what it finds.
- **Self-registration.** Adding a check needs no edit to a central dispatch function.
- **No hidden order.** Checks don't depend on running after another check. Facts that several checks share are worked out once, in an explicit, documented place, and read by the checks.
- **Folders by area.** Checks are grouped by area, one Go package per area.
- **Same findings.** For every recorded case, build_jsonld returns the same issues, with the same paths, severities and wording. It also returns the same counts, page elements and script tag.

## Non-goals

- No new checks, and no changed wording or severity. That belongs in other PRs. The product snippet checks from PR #12 move as they are.
- No change to the MCP tool's inputs or to the JSON shape of its output.
- No change to the public `check` package's API (PR #11): `check.Vocabulary` and `check.Errors` keep their behavior.

## Anti-goals

These would make the work a failure even if every test passed:

- A new god package: the old `checker` renamed, or a `Context` that grows a method for every check. Shared code is split by what it is about: the vocabulary, the draft, the page, plain text, and the audit framework.
- Copying one area's shared traversal into each of its check files. For example, the snippet checks' `@id` resolver and offers walk would be copied into each snippet check. Shared code for an area lives in one helper file of that area.
- Checks that still depend on run order through mutable state on the context.

## Constraints

- **Golden gate.** `internal/vocabtool/golden_test.go` replays 203 recorded `Check` and `Vocabulary` calls (`testdata/golden/cases.jsonl.gz`), recorded at commit `1e9de80`.
  - It compares the issues as a set.
  - It compares error and warning counts, page elements and script tag exactly.
  - It must pass after every commit. A step that changes it has a bug.
- **Existing end-to-end tests stay.** These tests stay as the integration layer: `lmb_test.go`, `audit_test.go` (including the ecommerce audit doc), `shop_test.go`, `vocabtool_test.go` and the others in `internal/vocabtool`.
  - The only edits allowed are to assertions that depend on issue order.
  - The new per-check tests are added beside them, not put in their place.
- **Commits.** Go 1.26 module. `go vet ./...` and `go test ./...` pass at every commit.

## Design

### Packages

| Package | Holds |
|---|---|
| `internal/vocabtool` | `Tool` and its lookups (TypeProperties, EnumValues, PropertyInfo), plus `Check` and `Vocabulary`. These two build the context, run the checks and assemble the `Report`. It imports every area package so their checks register. |
| `internal/vocabtool/vocab` | The vocabulary index that `Tool` builds today: classes, properties and members, applicable properties, enum-only properties, suggestable subtypes, did-you-mean suggestions. It imports no audit code, so tests can load it without a cycle. |
| `internal/vocabtool/jsonld` | The ordered JSON object (`Object`: decode, encode, get, set), the list of nodes with their paths, value helpers (`Values`, `Strs`, `Scalars`, `TypeNames`, …). It knows nothing of the vocabulary. |
| `internal/vocabtool/webpage` | Everything read from the rendered page: today's `pageFacts`, page kind, private, cards, sections, questions, tools, data blocks, option groups, details and the like, plus the HTML helpers. |
| `internal/vocabtool/txt` | Plain-string helpers the others share: normalized words, whole-word matching, clipping, URL normalization, money amounts. |
| `internal/vocabtool/audit` | The framework: `Normalize`, the `Check` interface, `Issue` and the severities, `Context`, the registry, `Run`, the shared ledgers (below) and memoized facts. |
| `internal/vocabtool/audit/audittest` | The harness that per-check tests use. |
| `internal/vocabtool/checks/<area>` | The checks, one per file. |

`Normalize` expands bare enum names to schema.org URLs. Today the vocabulary walk does this while it checks. `Normalize` does it as a pure pass with the same conditions:

- only known properties;
- not Google annotations such as `query-input`;
- not keys holding `:` or `/`;
- recursing only through property values and `@graph`, including untyped objects;
- only members of an enum-only range.

Areas, as a starting split. Area names never repeat a core package's name (`terms`, not `vocab`), so a check file never imports a package with its own name. The builder may move a check to a better-fitting area:

| Area | From today |
|---|---|
| `terms` | the vocabulary walk: `types`, `unknown`, `property`, `value` |
| `graph` | draft-only checks: `repeats`, `classTypes`, `namedTypes`, `webPageElements`, `bare`, `ownReviews`, `reviews`, `offers`, `personNames`, `pageNames` |
| `snippets` | `productSnippets` |
| `onpage` | `pageEntities`, `pageElements`, `shownText`, `anchors` |
| `content` | the rest of `content.go`'s page checks, and `figures.go` |
| `shop` | `shop.go`: page type, website, business, logo, social, catalog, listing, sort order |
| `product` | `product.go` |
| `policies` | `policies.go` |
| `costs` | `costs.go` and `prices` |
| `site` | `site.go` |
| `sources` | `sources.go` and `data.go` |
| `keyword` | `keyword.go`: keyword density, organizations, loose nodes |

### The interface

```go
// package audit
type Check interface {
	ID() string              // "<area>/<kebab-name>", stable, unique
	Run(ctx *Context) []Issue
}

func Register(c Check) // called from each check file's init()
```

- **One file per check.** A check file holds one exported struct, its `ID`, its `Run`, its doc comment, and the regexes and tables that only it uses. Its `init()` registers it.
- **Naming.** The file name is the snake_case of the ID's name, for example `checks/snippets/offer_count.go`, type `OfferCount`, ID `snippets/offer-count`.
- **Helper files.** Any other file in an area package is shared code for that area's checks, named for what it holds (for example `checks/snippets/offers.go`). It registers nothing.
- **Gating.** Each check guards its own `Run`: "needs the page", "only on a page search engines index", "only on a site that sells". It uses small `Context` predicates, so the condition is in the check's own file. Today the gating is spread through `content()`, `shop()`, `site()`, `figures()` and `costs()`, and each check must keep exactly the condition it has today.
- **`Issue` keeps its fields and JSON tags.** `vocabtool.Issue` becomes an alias of `audit.Issue`, and the severity constants are re-exported, so the MCP server and `check` package compile unchanged.

### Granularity

There is one check per distinct message template: per `c.add` format string, counting a template once when several call sites share it. Splitting must not change any message. Two cases are not split:

- a template built from a loop over fields stays one check: snippets' "Missing field review; Missing field aggregateRating";
- a template that only varies in a filled-in word stays one check.

This gives roughly 100 to 150 checks.

**The vocabulary walk.** The walk has early exits:

- an unknown type skips that type's superseded check;
- an unknown property, a Google annotation or a non-schema.org key skips that property's other checks and its value checks;
- only known properties' values are checked as enum members.

The `terms` area keeps those exits in one helper file, `checks/terms/walk.go`. It visits what the walk visits, in the order it does, and hands each check the node, path, key, term and property that today's code would check. Each term check is one template and uses that helper, so `Check` and `Vocabulary` see the same walk.

### Context

`Context` carries what the checks read:

- `Vocab`;
- the draft root and its node list;
- the page facts. These are `nil` with no page. "Known" facts carry only the page URL when there is no page, as `organizations`, `looseNodes` and `keyword` use today;
- the options: keyword, density and page URL.

It also holds a memo for facts computed once on first use: the draft index, the page's sections and elements, the product cards, and the two ledgers below.

`Context` methods stay generic: `Is`, `Single`, the page predicates. A helper used by one area lives in that area.

### Shared facts, made explicit

Two kinds of shared state decide which check reports something. Each becomes a ledger: a fact computed once, in a fixed order written down in one place in `audit`. Every check reads the ledger, so a check gives the same answer whether it runs alone or with the others.

**Price ownership.** Several checks report a price shown on the page, and the catch-all `prices` check reports what none of them did.

- **What a check takes.** A check takes exactly the amounts today's code claims for it: every `p.claim` and `p.claimed[f] = true` it makes. That includes amounts it claims whether or not it reports anything. Today `listing` claims every product card's prices and `shipping` claims its rates even when the draft holds them.
- In order, they are:
  1. listing;
  2. sale, extras and shipping, on a site that sells;
  3. data;
  4. breakdowns, rates, goods and estimates, on a site that sells nothing.
- Each later one skips the amounts an earlier one took.
- `prices` reports what is left.
- A check that takes prices implements a second interface beside `Check`. It returns the amounts it takes, given the amounts taken before it.
- The ledger lists those checks by ID in today's order and works out each one's "taken before" set once.
- The check's `Run` and its claims share one find function in its file.

**Retype ownership.** Several checks tell a node to take a more specific type, and a later one stays quiet on a node an earlier one already told.

- In order, they are:
  1. a property only a subtype takes (vocab walk);
  2. the Wikidata class (`classTypes`);
  3. the name or URL (`namedTypes`);
  4. the page's kind (`pageType`);
  5. the business (`business`);
  6. no premises (`onlineBusiness`).
- Today's rules carry over exactly:
  - `namedTypes` skips a node that sources 1 or 2 told, but its own suggestions block nothing;
  - `pageType` skips a node another source told, or one that `namedTypes` named with the same subtype;
  - `business` and `onlineBusiness` skip any node an earlier blocking source told.
- `pageType` stops reading issue messages. It asks the ledger.
- **Mechanism.** It mirrors the price ledger.
  - A retyping check implements a second interface beside `Check`. Given the ledger state before it, it returns the nodes it tells and the subtypes it names for each.
  - The state has two parts: the nodes a blocking source told, and every subtype named for each node by any source.
  - Whether a source blocks is fixed in the ledger's ordered list.
  - The check's `Run` and its retypes share one find function in its file, holding today's gating. For example, `onlineBusiness` only applies when the site sells nothing and shows no street address or opening time, and `business` only when the page has a document.
  - The first source, a property only a subtype takes, comes out of the vocabulary walk. It uses the `terms` area's walk helper.

The page elements report (`Report.PageElements`) is a memoized fact. `Check` copies it into the report, and the section checks report from it.

### Running and order

`Run` runs every registered check and gathers their issues. It then sorts them stably:

1. by severity: errors, then warnings, then info;
2. by check registration order. Go initializes packages sorted by import path and runs a package's `init` functions in file-name order, so this order is deterministic;
3. by the order the check emitted them.

The issue order changes from today. The issues do not.

`Vocabulary` normalizes, then runs only the `terms/` checks over the root as given. An array root keeps its `$[0]` paths, as today.

### Tests

`audittest` builds a context from a draft and optional page HTML, page text, page URL and keyword. It normalizes the draft the way `Check` does, runs one check, and returns that check's issues. The harness blank-imports every area so the ledgers see every check. Per-check tests are therefore external test packages (`package snippets_test`).

Each check's test file:

- is table-driven;
- has at least one case where the check fires, asserting path, severity and the full message;
- has at least one case where it stays quiet.

Where today's tests already hold a case for that check, the new test reuses its input.

Before the golden test is deleted, a meta-test in `internal/vocabtool` asserts that every registered check has a `<file>_test.go` beside its file in its area directory. It also asserts that every check ID is unique and matches its area's directory. The per-check firing case, together with the existing end-to-end tests (the golden cases were recorded from them, so the same inputs still run), is the safety net that stays.

When the split is done, the golden test and its data are deleted in the last commit. Its job ends with the refactor, and keeping it would mean regenerating 200 reports for every wording change. The per-check tests and the existing end-to-end tests remain.

### Migration

The order is strangler-style, sequential, with one commit per step. The golden test stays green at every commit.

1. **Core packages.**
   - Move the vocabulary index, the draft model, the text helpers and the page facts out to `vocab`, `jsonld`, `txt` and `webpage`.
   - Add `audit` with the interface, context, registry, run and ledgers.
   - Wrap what is left of the old `checker` as one temporary adapter check, so `Check` already runs through `audit.Run`.
2. **Areas, one at a time.** Move each area's checks to `checks/<area>`, one per file with tests, and delete them from the old `checker`.
   - Moving the price-taking and retyping checks switches them from the old mutable maps to the ledgers.
   - Until all such checks have moved, the adapter feeds the ledger. The builder settles the exact bridge.
3. **Finish.** When the adapter is empty, delete it and `checker`. Point `Vocabulary` at the `terms` group. Update the docs that describe the code layout: README, `guide.md` if it names files, and the server's description if it names files. Delete the golden test.

## Left to the builder

- **Names and APIs.** Exact type, method and helper names in `audit`, `jsonld`, `webpage` and `txt`, and the exported API of each.
- **The bridge.** How the adapter and the ledgers coexist during the migration.
- **Where checks go.** Which helper goes in which core package, and moving a check to a better area.
- **Fixtures.** Test-case HTML for page checks.

## Decisions made by the implementer, not the owner

- **Package names.** `jsonld`, `webpage` and `txt`, not `draft`, `page` and `text`. The code has dozens of locals named `draft`, `page` and `text`, which would shadow those package names. `terms` keeps the vocabulary walk's area apart from the core `vocab` package.
- **Base branch.** PR #11 and PR #12 merged together. Both are already on `refactor/audit-checks` (merge commit `0df924d`), so no rebase is needed, so `Vocabulary` (PR #11) and the snippet checks (PR #12) move too. Both PRs merge before this one.
- **Issue order.** Issues are sorted by severity, then check order. This differs from today's run order, but the issues themselves are the same.
  - Splitting checks makes some change of order unavoidable: issues group by check instead of by node.
  - The README, `guide.md` and the server description promise no order, and no test indexes issues by position.
  - Errors first helps the model reading the report.
- **Golden test.** It is temporary, and the last commit deletes it.
- **`pageType` and `namedTypes`.** `pageType` skips a node when any of `namedTypes`' subtypes for it is the one `pageType` wants. Today it substring-matches the first named subtype in the message. This counts as a design decision only if the golden shows no difference. If the golden differs, match today's behavior.
- **No `check` field.** Issue JSON gains no check-ID field, so the output shape stays as it was.
