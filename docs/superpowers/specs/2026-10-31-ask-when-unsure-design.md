# Ask when unsure: design

Date: 2026-10-31. Status: draft for review.

## Intent

schema-mcp makes quiet guesses about what things are. Hurricane Helene and
Fiesta San Antonio stay `Thing` because the class table never maps to Event.
"Car rental" is told to become an Organization because the checker cannot
tell a kind of business from one company. A shop name with "jewellery" in
it becomes a JewelryStore. Nobody sees these guesses as guesses.

The owner wants the MCP to stop guessing silently. When it is not sure what
something is, it asks the owner a multiple-choice question in Claude Code,
with a way to type an answer instead, and builds on the answer. The owner
would rather be asked too often than have a wrong guess slip through.

## Goals

- Every type decision the MCP makes carries a confidence score (0–100) and
  the reasons behind it.
- Below a threshold, the MCP asks the user in a dialog and uses the answer.
- Answers are remembered, so the same question is not asked twice.
- An `auto` mode keeps today's behaviour (guess, never ask) for headless and
  batch runs. The report lists every guess with its score and reasons.

## Non-goals

- No new data sources. The score uses what the MCP already fetches.
- No changes to how entities are chosen for a page (the relevance and
  similarity thresholds in select_entities). This is about what a chosen
  thing is, not whether it belongs on the page.
- No machine-learned model. The score is rule-based and explainable.

## Anti-goals

- A wrong answer is never silently kept. The user can see and change every
  remembered answer.
- A question about something obvious. Cost, Price and Truck stay Thing
  without a question: Wikidata marks them as kinds of thing and nothing
  says otherwise.
- The same question coming back on every page or every build_jsonld call
  once answered.
- A run that hangs waiting for an answer nobody can give: headless runs
  never block.

## Decided by the owner

- **Default mode is `ask`.** It can be set to `auto`.
- **Questions are multiple choice with an option to type.**
- **Answers are remembered.**
- **The decision uses a confidence score.**
- **Ask often.** The default threshold leans to asking: a guess without
  agreeing evidence is asked.

## How it works

### Modes and settings

A new section in the existing settings file
(`~/.config/go-json-schema/schema-mcp.yml`, see
`cmd/schema-mcp/config.go` and `schema-mcp.yml.dist`):

```yaml
questions:
  mode: ask          # ask | auto
  threshold: 80      # ask below this score; tuned per "Calibration" below
  answers_file: ""   # default ~/.config/go-json-schema/answers.yml
```

- A flag and env var override `mode` for one run (`-questions auto`,
  `SCHEMA_MCP_QUESTIONS=auto`), matching the existing flag-plus-env pattern
  in `cmd/schema-mcp/main.go`.
- In `ask` mode, if the client did not declare the elicitation capability,
  the call runs as `auto`. The report says so.
- If the user declines or cancels the dialog, the call also runs as `auto`.
- Headless Claude Code (`claude -p`) cancels elicitation requests, so it
  falls back the same way.

### Where the MCP asks

Questions are asked during these tool calls:

- **select_entities** (`internal/research`): the type of each chosen
  entity. Today `classtype.Of` sets it (`selection_identity.go` and
  `entities.go`), and "" means Thing. The answer sets the type on the
  entity node the tool returns.
- **resolve_entities**: the same type decision for the entities it
  returns. Also a phrase that lands on a Wikipedia disambiguation page
  (`StatusAmbiguous`, with `Candidates`). The question is which candidate
  the phrase means, with "none of these" as an option.
- **build_jsonld** (`internal/vocabtool`): the checks that end in "type it
  X, unless …" or choose between readings:
  - `classTypes` and `namedTypes` (`subtypes.go`): a node's subtype from
    its Wikidata class, or from the words in its name or URL;
  - `storeKinds` (`shop.go`): a shop's Store subtype from its name;
  - `onlineBusiness` (`site.go`): an online-only business versus one with
    premises;
  - `estimates` and the price fallbacks (`costs.go`, `content.go`): the
    business's own price (an Offer) versus what others charge (a
    MonetaryAmount).

  build_jsonld keeps reporting only; it never rewrites the draft.
  (Builder's call, so the tool's contract stays the same.) An answer
  rewrites the warning to state the user's answer and the exact change, for
  example: "you said Hurricane Helene is an Event: type it Event". It stays
  a warning, not an error. If the draft already matches the answer, the
  warning is dropped. A remembered answer that the draft contradicts gets
  the same rewritten warning.

### The confidence score

Every decision gets a score from 0 to 100 and a list of reasons in plain
words. Evidence for entity types:

- **For the guess:**
  - a Wikidata P31 class maps in the class table (strongest);
  - the TextRazor DBpedia type of the same entity maps to the same
    schema.org type (`textrazor.Entity.Types`, already fetched for
    page-linked entities, not used for typing yet);
  - several P31 classes agree;
  - the page's words fit (the node's heading or name names the type).
- **Against the guess:**
  - P31 classes that map to different types (today the first table row
    wins silently);
  - Wikidata and DBpedia disagree;
  - the item has P279 (subclass of), so it may be a kind of thing, while
    its P31 maps to a type that is not one of the always-a-class types
    (`Kinds` in `internal/classtype`). Example: moving company, "instance
    of company" but "subclass of business".
- **Unknown, scored low:** no clue at all on an item with no P279. That is
  an item that looks like one specific thing with no type, like Hurricane
  Helene.
- **Confident Thing, no question:** the item has P279 and no P31 class maps
  to any type, like Cost or Truck.
- **A remembered answer scores 100.**

The build_jsonld guesses are scored by how many independent clues back
them. For online-only, those clues are: no street address, no opening
hours, and shipping or delivery copy.

The weights and the threshold are the builder's to set (see Calibration).
The reasons must name the evidence, for example: "35: Wikidata class
hurricane has no schema.org type; DBpedia says Event".

### The question

One dialog per tool call holds all of that call's questions. Each item gets
two fields, because a form field is either a pick-list or a text box, not
both (MCP spec 2025-11-25, form mode: flat object, primitive fields):

- **A titled single-select list.** The likely answers, most likely first.
  For a type question: the candidate types the evidence names, `Thing`
  ("a kind of thing, or nothing more specific"), "Something else (type
  below)" and "Skip (use the guess)".
- **A text box under it.** It is read when "Something else" is picked. The
  typed value must be a schema.org type. If it is not, the MCP asks again
  with the closest real type names.

The field description shows the evidence: the Wikidata description and
classes, the DBpedia type, and the score and reasons.

"Skip" uses the guess and is not written to the answers file.

To stop skipped items from coming back on every call (builder's call):

- **Skip:** the item is muted for the rest of the server session (the
  server process). A later session asks again.
- **Decline:** mutes every item in that dialog for the session.
- **Cancel (Esc):** runs that call as `auto` only. The items are asked
  again on the next call.

A typed answer that is not a schema.org type is asked once more (builder's
call), with the closest real type names as options. Cancelling that second
dialog skips the item. The other answers from the first dialog stand.

The go-sdk version in go.mod (v1.8.0) has `ServerSession.Elicit`. Claude
Code declares elicitation `form` and `url` support and shows form requests
as a terminal dialog.

### Remembered answers

A plain YAML file the user can read and edit. It is user data, not cache:
it is not under the `-cache` directory, and `-cache ""` does not disable
it.

- **Facts about a thing** are keyed by Wikidata QID and apply on every
  site. Example: Q-id of Hurricane Helene → Event.
- **Facts about a business** are keyed by site (the host of the page URL or
  the WebSite url) and apply only there. Examples: "online only", "these
  prices are our own", Store subtype.
- **Facts with no QID** are keyed by site plus a name (builder's call):
  - a node's name-based subtype (`namedTypes`) or Store subtype
    (`storeKinds`): site + the node's name;
  - an entity with no Wikidata item: site + its name;
  - an ambiguous phrase (`StatusAmbiguous`): site + language + the phrase,
    because "Mercury" can mean different things on different sites.
- Each entry records the answer, the question as asked, and the date.

A new tool, `answers`, lists the remembered answers and forgets one (by
QID or site key), so the question is asked again next time. Editing the
file by hand works too.

### Report in auto mode

Every guess under the threshold is listed in the tool result, under a
`guesses` field: what was decided, the score, the reasons and the options
it would have asked. This lets a headless run be reviewed later. It also
lets Claude ask the user itself when the client cannot show a dialog.

## Calibration

Use the frozen Lift My Boxes pages
(`marketing-research/schema-mcp-gaps/html`, 121 pages, read-only) and the
Queen of Diamonds pages (`internal/vocabtool/testdata/shop`). Measure
questions per page with an empty answers file and the default threshold.

The owner said "ask often". So the default asks about every decision that
lacks agreeing evidence. Two cases need no question:

- confident kinds of thing: the item has P279 and no P31 class maps;
- confident types: Wikidata and DBpedia agree, or a remembered answer
  exists.

For scale: the median Lift My Boxes page carries 13 untyped Wikidata
entities, and the site has 272 distinct ones. Most of these are kinds of
thing (Cost, Price, Truck) and fall in the first no-question case.

Pass or fail:

- the four named cases are asked on a first run;
- Cost, Price and Truck are not asked;
- a second run with the first run's answers remembered asks nothing
  already answered;
- the first-run question count per page stays within the owner's limit
  (see Open question).

Report the per-page question counts before fixing the threshold:

- first run;
- second run with the first run's answers remembered;
- which items were asked.

The cases named in the gap review must all be asked on a first run:
Hurricane Helene, Fiesta San Antonio, Car rental, Moving company.

## Open question

- How many questions the owner will accept on the first page of a new site.
  This sets the default threshold and whether a dialog splits.

## Constraints

- Go, inside the existing module. Keep the current checks' behaviour in
  `auto` mode. Every existing test stays green, including
  `TestLMBAcceptance` (48/48).
- Tests must not touch the user's real answers file or settings. Use a temp
  answers file, and an in-memory client with an elicitation handler for the
  dialog paths.
- No network in tests beyond what existing tests already stub.

## Left to the builder

- Score weights and the default threshold, within "ask often" and the
  calibration above.
- Option order and wording in the dialog.
- How many items one dialog holds before it splits into several.
- The exact shape of the `guesses` field and of the answers file.
- How concurrent tool calls share one answers file safely.
- Which branch to build on. The class-table and checker changes this
  builds on are on `feat/ecommerce-schema-report`, uncommitted.
