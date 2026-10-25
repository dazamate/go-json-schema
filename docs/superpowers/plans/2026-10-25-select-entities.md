# select_entities Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the `select_entities` MCP tool, which reads our own page and its keyword and returns the `about` and `mentions` entities as ready schema.org `Thing` nodes, chosen only from measured data.

**Architecture:** `research.SelectEntities` reads the page through ChromiumFish, links it with TextRazor, looks the keyword up as exact Wikipedia titles, checks every identity by QID on Wikidata, scores TextRazor's themes with a local bge-base-en-v1.5 model, and adds Bing IDs from Azure AI Language. The two new outside dependencies sit behind interfaces in `research` (`ThemeScorer`, `BingLinker`): the new packages `internal/embed` and `internal/azurelang` implement them, and the tests replace them with fakes. Wikipedia and Wikidata are reached only through the browser, as `resolve_entities` does today.

**Tech Stack:**
- Go 1.26 module and the MCP go-sdk v1.8.0 (existing).
- `github.com/yalue/onnxruntime_go` v1.38.0 (cgo; ONNX Runtime C API 1.31.0).
- `github.com/sugarme/tokenizer` v0.3.0 (pure Go; reads `tokenizer.json`).
- ONNX Runtime 1.31.0 shared library, downloaded on first use.
- Python only to record reference fixtures, using google-seo-docs' venv.

**Spec:** `docs/superpowers/specs/2026-10-25-own-page-entities-design.md`. The calibration data is in `docs/superpowers/specs/2026-10-25-own-page-entities/`. Read the spec first: this plan argues from it and does not repeat its reasons.

## Global Constraints

**Defaults:**
- The defaults are `min_relevance` 0.3, `min_topic` 0.5, `min_keyword_similarity` 0.40 and `min_page_similarity` 0.55.
- The tool description and the README must say they are our judgement, set by eye from one calibration page, not from any Google source.

**Selection rules:**
- `about` and `mentions` have no caps.
- Every `mentions` entity is named in the page's visible text.
- Never search by name. Don't use `list=search`, `srsearch`, `wbsearchentities` or any other search. The keyword's exact-title lookups are the only name-based step.
- IDs come only from Google (P646 `/m/`, P2671 `/g/`), Bing (Azure `bingId`), Wikipedia, Wikidata and DBpedia. DBpedia is used for `en` pages only.
- No `@id` on entity nodes. Never use `synonymProps` borrowing in this tool.
- The Google link is `https://www.google.com/search?kgmid=<mid>`, everywhere.

**Outside calls:**
- Every Wikipedia and Wikidata call goes through the browser with `fetchJS` from a Wikipedia tab. Wikimedia returns 403 to Go's HTTP client.
- TextRazor and Azure receive only text the browser already fetched.

**Secrets:**
- The TextRazor key and the Azure key are never logged, never put in an error and never written to a fixture.

**Tests and builds:**
- The default `go test ./...` makes no live calls. Tests that do are gated:
  - `SCHEMA_MCP_LIVE=1` for the real browser;
  - `SCHEMA_MCP_RECORD=1` to re-record fixtures;
  - `SCHEMA_MCP_MODEL=1` for the real model.
- `CGO_ENABLED=0 go build ./...` must keep building. Without cgo the tool runs with no themes and a note.
- `page_terms`, `resolve_entities` and `serp` keep their behaviour. The only existing tests to edit are the ones this plan names.
- Match the surrounding code's comment density, naming and idiom.

**Executor environment:**
- `cat`, `head`, `tail` and `wc` are blocked in Bash: read files with the Read tool.
- Run the full test suite through context-mode's `ctx_batch_execute`.
- `chromiumfish` is on PATH. The venv python is `/home/dazamate/projects/google-seo-docs/.venv/bin/python`.

**Commits:**
- The working tree already holds earlier uncommitted work. Ask the user before the first commit.
- Stage only the files the task lists, never `git add -A`.
- End each message with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`.

## Review Focus

The spec doesn't mention these five inputs, and none of the spec's tests cover them. They are the most likely to bite in use. Each one has a test in the task named:

1. **The first model download is interrupted, or two calls start it at once.** No corrupt or partial file stays in the cache, each file is fetched once, and a bad hash is an error. Tested in Task 6.
2. **A keyword with function words**, such as "cost to move a piano" or "movers near me". The stop words are never looked up on their own, a run never starts or ends with one, and the focus check skips them. Without this rule, "a" resolves to the article on the letter A. This rule is my call: see Task 8.
3. **The page's language is not plain `en`** (`en-US`, empty, `de`). The edition is the part before `-`, else `en`. A non-English page gets no DBpedia link, and Azure is skipped with a note unless the language is `en` or `es`. Tested in Tasks 9 and 10.
4. **Own-site links when hosts differ in form:**
   - the page was reached through a redirect, so links are judged against `final_url`;
   - `www.` and the bare host count as the same site;
   - a link to another host never counts;
   - a link to the page itself or to the site root never counts.
   Tested in Task 10.
5. **Wikipedia link queries larger than one response.** `prop=links` answers that carry `continue` are followed until done, and titles are batched 50 at a time. No link is lost. Tested in Task 8.

## Files

| File | Responsibility |
|---|---|
| `schema/entity.go` (modify) | `KGSearch(mid)`; `Entity.KGMIDs`; `WikidataID`, `BingEntityID` PropertyValues |
| `cmd/schema-mcp/config.go`, `schema-mcp.yml.dist`, `main.go` (modify) | `azure_language` settings; wiring of `Bing` and `Themes` |
| `internal/azurelang/azurelang.go` (create) | Azure AI Language entity-linking client |
| `internal/embed/embed.go` (create) | Recipe: windows, mean, cosine, `Scorer` |
| `internal/embed/download.go` (create) | Hash-pinned, atomic first-use downloads |
| `internal/embed/onnx.go` (create, `cgo`), `onnx_nocgo.go` (create, `!cgo`) | bge-base ONNX encoder |
| `internal/embed/testdata/` (create) | `record_reference.py`, `reference.json`, `page.txt` |
| `internal/research/pages.go`, `linking.go`, `google.go`, `serp.go` (modify) | Main-content links in `pageJS`; shared `analyze`; exported `Progress`; new `Researcher` fields |
| `internal/research/selection_keywords.go` (create) | Words, stems, stop words, focus, keyword lookups |
| `internal/research/selection_identity.go` (create) | Items by QID, identity check, entity node, Bing attach |
| `internal/research/selection.go` (create) | Types, `SelectEntities`, gate, themes, roles |
| `internal/research/testdata/select/` (create) | `page.html`, `main.txt`, `textrazor.json`, `cosines.json`, `wiki.json` |
| `internal/mcpserver/server.go`, `guide.md` (modify); `README.md` (modify) | Tool, instructions, prompt, guide, docs |

---

### Task 1: Reference fixtures from the Python recipe

No Go code. Every later similarity check is measured against these files. They are recorded at full precision, without rounding.

**Files:**
- Create: `internal/embed/testdata/record_reference.py`, `internal/embed/testdata/page.txt`, `internal/embed/testdata/reference.json`
- Create: `internal/research/testdata/select/page.html`, `main.txt`, `textrazor.json`, `cosines.json`

**Interfaces:**
- Produces `reference.json`:
  - `{"model", "source", "revision", "fastembed", "onnxruntime", "keyword"}`;
  - `"query": {"text", "ids"}`;
  - `"windows": [{"text", "ids"}]`;
  - `"probes": [{"text", "ids"}]`;
  - `"themes": [{"label", "ids", "keyword", "page"}]`.
- Produces `cosines.json`: `{"model", "keyword", "themes": {"<label>": {"keyword": f, "page": f}}}`.

- [ ] **Step 1: Copy the calibration inputs**

```bash
C=docs/superpowers/specs/2026-10-25-own-page-entities
mkdir -p internal/embed/testdata internal/research/testdata/select
cp $C/main.txt internal/embed/testdata/page.txt
cp $C/page.html $C/main.txt internal/research/testdata/select/
cp $C/tr_page.json internal/research/testdata/select/textrazor.json
```

- [ ] **Step 2: Write `internal/embed/testdata/record_reference.py`**

```python
"""Records what the Go embedder must reproduce: google-seo-docs' recipe
(seoaudit/embed.py, fastembed with Qdrant/bge-base-en-v1.5-onnx-Q) on the
calibration page, at full precision. Re-run only when the calibration data
or the model changes:

  /home/dazamate/projects/google-seo-docs/.venv/bin/python -I internal/embed/testdata/record_reference.py
"""
import json
import sys
from importlib.metadata import version
from pathlib import Path

sys.path.insert(0, "/home/dazamate/projects/google-seo-docs")
from seoaudit.embed import QUERY_INSTRUCTION, Embedder, chunks, mean_vector  # noqa: E402

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
KEYWORD = "moving cost calculator"
PROBES = ["Café crème brûlée, naïve façade", " ".join(["moving"] * 600)]

page = (HERE / "page.txt").read_text()
tr = json.loads((ROOT / "internal/research/testdata/select/textrazor.json").read_text())["response"]
labels = sorted({t["label"] for t in tr["topics"] if t.get("wikidataId")})
emb = Embedder()
tok = emb._load().model.tokenizer
ids = lambda text: tok.encode(text).ids
dot = lambda a, b: sum(x * y for x, y in zip(a, b))

query = QUERY_INSTRUCTION + KEYWORD
kv = emb.embed([query])[0]
windows = chunks(page)
pv = mean_vector(emb.embed(windows))
themes = []
for label in labels:
    v = emb.embed([label])[0]
    themes.append({"label": label, "ids": ids(label), "keyword": dot(kv, v), "page": dot(pv, v)})
ref = {"model": "BAAI/bge-base-en-v1.5", "source": "Qdrant/bge-base-en-v1.5-onnx-Q",
       "revision": "199291fdd1aa89faf9c20b722dc72ad5e17aa0d0", "fastembed": version("fastembed"),
       "onnxruntime": version("onnxruntime"), "keyword": KEYWORD,
       "query": {"text": query, "ids": ids(query)},
       "windows": [{"text": w, "ids": ids(w)} for w in windows],
       "probes": [{"text": p, "ids": ids(p)} for p in PROBES], "themes": themes}
(HERE / "reference.json").write_text(json.dumps(ref, indent=1, ensure_ascii=False) + "\n")
cos = {"model": ref["model"], "keyword": KEYWORD,
       "themes": {t["label"]: {"keyword": t["keyword"], "page": t["page"]} for t in themes}}
(ROOT / "internal/research/testdata/select/cosines.json").write_text(json.dumps(cos, indent=1, sort_keys=True) + "\n")
print(f"{len(windows)} windows, {len(themes)} themes")
```

- [ ] **Step 3: Run it**

Run: `/home/dazamate/projects/google-seo-docs/.venv/bin/python -I internal/embed/testdata/record_reference.py`

Expected:
- the script prints `7 windows, 69 themes` (`page.txt` has 963 words);
- `reference.json` shows `"onnxruntime": "1.31.0"`;
- Freight transport's `keyword` value is `0.40009…`;
- Accounting's `page` value is `0.55186…`.

- [ ] **Step 4: Check that no API key is in a fixture**

Run a python snippet that reads `api_key` from `~/.config/go-json-schema/schema-mcp.yml` by regex, never printing it. It searches every file under `internal/*/testdata/` for the key and prints only `no key in fixtures`, or `KEY FOUND: <file>`.

Expected: `no key in fixtures`.

- [ ] **Step 5: Commit** (ask the user first; see Global Constraints)

```bash
git add internal/embed/testdata internal/research/testdata/select
git commit -m "test: record bge-base reference and select_entities calibration fixtures"
```

---

### Task 2: Plain kgmid link and the schema helpers

**Files:**
- Modify: `schema/entity.go`, `schema/entity_test.go:12,39,57`, `schema/example_test.go:210,245`, `schema/doc.go:43` (only if the wording names the old form), `README.md:150`, `internal/mcpserver/guide.md:46`

**Interfaces:**
- Produces:
  - `func KGSearch(mid string) URL`. The query argument is removed. This changes the public API, but a grep of `/home/dazamate/projects` on 2026-10-25 found no `KGSearch(` caller outside this repo.
  - `Entity.KGMIDs []string`: more Knowledge Graph IDs, each added as a sameAs link and a GoogleMachineID after `KGMID`.
  - `func WikidataID(qid string) *PropertyValue` gives `{"propertyID":"Wikidata","value":qid}`.
  - `func BingEntityID(id string) *PropertyValue` gives `{"propertyID":"BingEntityID","value":id}`.

- [ ] **Step 1: Write the failing tests** in `schema/entity_test.go`
  - `TestIdentityURLHelpers`: `KGSearch("/g/122hcnps")` gives `"https://www.google.com/search?kgmid=/g/122hcnps"`.
  - `TestEntityThingCarriesEverySource` and `TestEntityPerSourceSplitsOneThingPerURL`: in the `want` strings, every `?q=…&kponly&kgmid=` becomes `?kgmid=`.
  - `TestEntityThingAddsEveryKGMID`: `Entity{Name: "x", KGMID: "/m/1", KGMIDs: []string{"/g/2"}}.Thing()` marshals to sameAs `["https://www.google.com/search?kgmid=/m/1","https://www.google.com/search?kgmid=/g/2"]`, with identifier `[GoogleMachineID kg:/m/1, GoogleMachineID kg:/g/2]` in that order.
  - `TestWikidataAndBingIdentifiers`: the two helpers marshal to `{"@type":"PropertyValue","propertyID":"Wikidata","value":"Q31087"}` and `{"@type":"PropertyValue","propertyID":"BingEntityID","value":"a093e9b9-90f5-a3d5-c4b8-5855e1b01f85"}`.
- [ ] **Step 2: Run them to see them fail.** Run `go test ./schema/`. Expected: a compile error, because `KGSearch` takes 2 arguments.
- [ ] **Step 3: Implement.**
  - `KGSearch` returns `"https://www.google.com/search?kgmid=" + mid`, unescaped. That is the form verified to show a panel.
  - `SameAsURLs` and `Thing` loop over `KGMID` followed by `KGMIDs`.
  - Update the expected output in the `Example` comments (`example_test.go:210,245`), README line 150, and guide line 46 (`- https://www.google.com/search?kgmid=/m/… or /g/… (only when known)`).
  - Leave `schema/node_test.go:58` alone: it tests URL encoding, not `KGSearch`.
- [ ] **Step 4: Run `go test ./schema/ ./internal/...`.** Expected: PASS. The `internal/research` resolver tests build `Thing` through `schema.Entity` and still pass.
- [ ] **Step 5: Commit** with the message `fix(schema): use the plain kgmid Google link`.

---

### Task 3: `azure_language` settings

**Files:**
- Modify: `cmd/schema-mcp/config.go`, `cmd/schema-mcp/config_test.go`, `cmd/schema-mcp/schema-mcp.yml.dist`

**Interfaces:**
- Produces `Config.AzureLanguage AzureLanguageConfig` with the yaml key `azure_language`, holding `Endpoint` (`endpoint`) and `APIKey` (`api_key`).

- [ ] **Step 1: Write the failing tests**
  - `TestLoadConfigReadsTheAzureSettings` gets endpoint `https://x.cognitiveservices.azure.com/` and key `k1`.
  - `TestLoadConfigRejectsHalfAnAzureSetting`: an endpoint without a key, or a key without an endpoint, gives an error naming `azure_language`. The error doesn't contain the key's value.
  - `TestLoadConfigRejectsAPlainHTTPAzureEndpoint`: `http://` gives an error.
  - `TestExampleConfigLoads` still passes with the new `.dist` block.
- [ ] **Step 2: Run `go test ./cmd/schema-mcp/`.** Expected: FAIL, because the `azure_language` field is unknown.
- [ ] **Step 3: Implement.**
  - Validate in `loadConfig` after decoding. Both fields are empty or both are set, and the endpoint parses with scheme `https`.
  - Add a `.dist` block like the `textrazor` one, with empty values. Its comment says:
    - it adds Bing entity IDs to select_entities through Azure AI Language entity linking;
    - the free F0 tier exists;
    - entity linking retires on 1 September 2028;
    - only page text already fetched is sent.
- [ ] **Step 4: Run `go test ./cmd/schema-mcp/`.** Expected: PASS.
- [ ] **Step 5: Commit** with the message `feat(config): add azure_language settings`.

---

### Task 4: Azure AI Language entity-linking client

**Files:**
- Create: `internal/azurelang/azurelang.go`, `internal/azurelang/azurelang_test.go`

**Interfaces:**
- Produces:
  - `const APIVersion = "2022-05-01"`, `MaxDocChars = 5120`, `MaxDocsPerRequest = 5`, `RecordChars = 1000`.
  - `type Client struct { Endpoint, Key string; HTTP *http.Client }`. The default `HTTP` client has a 90 s timeout.
  - `type Entity struct { BingID, Name, ID, URL, DataSource, Language string }`, with the JSON tags `bingId`, `name`, `id`, `url`, `dataSource` and `language`.
  - `type Result struct { Entities []Entity; Records int }`.
  - `func (c *Client) Link(ctx context.Context, text, language string) (Result, error)`.
  - `func Documents(text string) []string`.
  - `func Records(docs []string) int`: the sum over docs of ⌈runes/1000⌉.
  - `func Supported(language string) bool`: true for `en` and `es` only.

**Request format:**
- `POST {Endpoint without trailing /}/language/:analyze-text?api-version=2022-05-01`.
- Headers: `Ocp-Apim-Subscription-Key: <Key>` and `Content-Type: application/json`.
- Body: `{"kind":"EntityLinking","parameters":{"modelVersion":"latest"},"analysisInput":{"documents":[{"id":"1","language":"en","text":"…"}]}}`.
- Document ids run `"1"…"n"` across all requests, and requests are sent one after another.
- The response shape is the quickstart's (`results.documents[].entities[]`, `results.errors[]`).

- [ ] **Step 1: Write the failing tests.** Serve the quickstart's documented sample response from `httptest`, with the Microsoft, Bill Gates, Paul Allen and April 4 entities and bingId `a093e9b9-90f5-a3d5-c4b8-5855e1b01f85` for Microsoft.
  - `TestLinkPostsDocumentsAndDecodesEntities`: checks the path, the `api-version`, the key header, the body's `kind` and `language`, and decodes 4 entities with Microsoft's `bingId` and `url`.
  - `TestDocumentsSplitsAtLineBreaks`:
    - 3 paragraphs of 3,000 runes give 3 documents;
    - 2 paragraphs of 2,000 runes give 1 document;
    - one 12,000-rune line with spaces gives 3 documents, each ≤ 5,120 runes, cut at spaces;
    - joining the documents' words gives the input's words.
  - `TestLinkSendsAtMostFiveDocumentsPerRequest`: 7 documents produce 2 requests (5 and 2), and the entities from both are merged.
  - `TestRecordsCountsThousandCharacterBlocks`: documents of 1, 1,000, 1,001 and 5,120 runes count 1, 1, 2 and 6 records.
  - `TestLinkErrorNeverEchoesTheKey`: on HTTP 401 the error contains neither the key nor the endpoint's query string.
  - `TestLinkReportsDocumentErrors`: a non-empty `results.errors` gives an error naming the document id.
  - `TestSupportedLanguages`: `en` and `es` are true; `de` and `""` are false.
- [ ] **Step 2: Run `go test ./internal/azurelang/`.** Expected: FAIL, because the package doesn't exist yet.
- [ ] **Step 3: Implement.** HTTP errors trim the body to 300 bytes, as `textrazor.Analyze` does. `Documents` packs paragraphs greedily:

```go
// Documents splits text at line breaks into documents of at most MaxDocChars
// characters (runes); a line longer than that is cut at spaces.
func Documents(text string) []string {
	var docs []string
	var cur []string
	n := 0
	flush := func() {
		if len(cur) > 0 {
			docs = append(docs, strings.Join(cur, "\n"))
			cur, n = nil, 0
		}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		for utf8.RuneCountInString(line) > MaxDocChars {
			flush()
			cut := cutAtSpace(line, MaxDocChars) // byte index of the last space within MaxDocChars runes, else of the rune limit
			docs = append(docs, strings.TrimSpace(line[:cut]))
			line = strings.TrimSpace(line[cut:])
		}
		if line == "" {
			continue
		}
		if l := utf8.RuneCountInString(line); n > 0 && n+l > MaxDocChars { // n counts each line plus its "\n"
			flush()
		}
		cur = append(cur, line)
		n += utf8.RuneCountInString(line) + 1
	}
	flush()
	return docs
}
```

- [ ] **Step 4: Run `go test ./internal/azurelang/`.** Expected: PASS.
- [ ] **Step 5: Commit** with the message `feat(azurelang): add Azure entity-linking client for Bing IDs`.

---

### Task 5: The similarity recipe

**Files:**
- Create: `internal/embed/embed.go`, `internal/embed/embed_test.go`

**Interfaces:**
- Produces:
  - `const Model = "BAAI/bge-base-en-v1.5 (Qdrant/bge-base-en-v1.5-onnx-Q@199291f)"`.
  - `QueryInstruction = "Represent this sentence for searching relevant passages: "`.
  - `WindowWords = 200`, `WindowStride = 150`, `MaxWindows = 40`, `MaxTokens = 512`.
  - `type Encoder interface { Encode(ctx context.Context, text string) ([]float64, error) }`. It returns a unit vector.
  - `func Windows(text string) []string`.
  - `func Mean(vs [][]float64) []float64`: the normalised mean.
  - `func Cosine(a, b []float64) float64`: unrounded.
  - `type Scorer struct { Enc Encoder }`.
  - `func (s *Scorer) Model() string`.
  - `func (s *Scorer) Score(ctx context.Context, keyword, page string, labels []string) (kw, pg []float64, err error)`. It embeds the keyword with `QueryInstruction`, embeds the page as `Mean` of its `Windows`, and embeds each label alone with no instruction. A page with no words is an error.

- [ ] **Step 1: Write the failing tests**
  - `TestWindowsMatchesTheReference`: `Windows(testdata/page.txt)` equals the `text` of each entry in `reference.json`'s `windows`, in order. This is a parity check that needs no model.
  - `TestWindowsOverlapAndLimit`:
    - 450 words give windows starting at words 0, 150 and 300;
    - 10 words give 1 window;
    - 50,000 words give 40 windows;
    - `""` gives none.
  - `TestMeanNormalises`: `Mean({{1,0},{0,1}})` is `{0.7071…, 0.7071…}`.
  - `TestScoreUsesTheRecipe`: the fake encoder records its inputs. The keyword goes in with the instruction; each window and each label goes in bare. `kw[i]` is the cosine against the query and `pg[i]` the cosine against the mean.
  - `TestScoreRejectsAnEmptyPage`.
- [ ] **Step 2: Run `go test ./internal/embed/`.** Expected: FAIL, because the package doesn't exist yet.
- [ ] **Step 3: Implement.** `Windows` is `seoaudit.embed.chunks` exactly: split on whitespace, start at `i := 0` and step by 150 while `i < max(1, len(words)-200+150)`, join `words[i:min(i+200, len(words))]` with single spaces, and keep the first 40.
- [ ] **Step 4: Run `go test ./internal/embed/`.** Expected: PASS.
- [ ] **Step 5: Commit** with the message `feat(embed): add the bge similarity recipe`.

---

### Task 6: The bge-base ONNX encoder, its downloads and the parity check

**Files:**
- Create: `internal/embed/download.go`, `internal/embed/download_test.go`, `internal/embed/onnx.go` (`//go:build cgo`), `internal/embed/onnx_nocgo.go` (`//go:build !cgo`), `internal/embed/onnx_test.go` (`//go:build cgo`)
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes `Encoder`, `Scorer` and `MaxTokens` from Task 5.
- Produces:
  - `func NewONNX(dir string, progress func(ctx context.Context, msg string)) *ONNX`. It is lazy: nothing loads until the first `Encode`. `progress` may be nil.
  - `func (o *ONNX) Encode(ctx context.Context, text string) ([]float64, error)`.
  - `func (o *ONNX) Close() error`.
  - Unexported `func (o *ONNX) tokens(ctx context.Context, text string) ([]int, error)` for the parity test.
  - Unexported `ensure(ctx context.Context, dir string, as []asset, progress func(string)) error`. It holds a package mutex, so concurrent calls download each file once.

**Pinned assets.** `dir` is `<UserCacheDir>/go-json-schema/models`.

| Asset | URL | SHA-256 | Saved as |
|---|---|---|---|
| model | `https://huggingface.co/Qdrant/bge-base-en-v1.5-onnx-Q/resolve/199291fdd1aa89faf9c20b722dc72ad5e17aa0d0/model_optimized.onnx` | `4e556722bc4f65716c544c8a931f1e90fb3f866e5741fd93a96f051d673339c7` | `bge-base-en-v1.5-onnx-Q@199291f/model_optimized.onnx` |
| tokenizer | same base URL, `/tokenizer.json` | `d241a60d5e8f04cc1b2b3e9ef7a4921b27bf526d9f6050ab90f9267a1f9e5c66` | `bge-base-en-v1.5-onnx-Q@199291f/tokenizer.json` |
| runtime (linux/amd64 only) | `https://github.com/microsoft/onnxruntime/releases/download/v1.31.0/onnxruntime-linux-x64-1.31.0.tgz` (SHA-256 `cc5c72baf5ae5c8238a6841f0897227be2d02826b9cf98eaf02fdefaeeb03a57`) | member `onnxruntime-linux-x64-1.31.0/lib/libonnxruntime.so.1.31.0`: `469849977cb60744ea160927e1a3a0643fd4c55c0aff743251c6d293402e334e` | `onnxruntime-1.31.0/libonnxruntime.so.1.31.0` |

**Model facts, checked on 2026-10-25:**
- Inputs `input_ids`, `attention_mask` and `token_type_ids` are int64 `[1, n]`. The output is `last_hidden_state` `[1, n, 768]`.
- The pooling is fastembed's: take token 0 (CLS) and L2-normalise in float64.
- Truncate to 512 tokens, longest-first, from the right. Use no padding: encode one text per run.
- Batching makes no difference. google-seo-docs embeds the 7 windows in one padded batch of 16. Embedding each window alone gave identical vectors (largest difference 0.0) and the same page cosines to 6 decimals. Checked on 2026-10-25.

- [ ] **Step 1: Write the failing download tests** (no cgo needed), against an `httptest` server.
  - `TestEnsureDownloadsAndVerifies`: the file lands in `dir` with the right bytes.
  - `TestEnsureRejectsABadHashAndLeavesNoFile`: the error names the asset, and neither the destination nor a `.part` or temp file is left in `dir`.
  - `TestEnsureSkipsAVerifiedFile`: a second call makes no request.
  - `TestEnsureReplacesACorruptFile`: a file whose hash is wrong is fetched again.
  - `TestEnsureDownloadsOnceForConcurrentCallers`: 4 goroutines run against a request-counting server, and each asset is requested exactly once.
  - `TestEnsureExtractsATarMember`: a tgz built in the test has its member extracted and verified, and the tgz is removed.
  - `TestEnsureReportsProgress`: at least one message names the size.
- [ ] **Step 2: Run `go test ./internal/embed/ -run Ensure`.** Expected: FAIL.
- [ ] **Step 3: Implement `download.go`.**
  - Hash while copying into `os.CreateTemp(dir, name+".*.part")`, compare, then `os.Rename`, and remove the temp file on every error path.
  - The HTTP client has no overall timeout and stops when `ctx` is cancelled.
  - Report progress at the start and about every 10 %.
- [ ] **Step 4: Run `go test ./internal/embed/ -run Ensure`.** Expected: PASS.
- [ ] **Step 5: Write the gated parity test** `TestONNXMatchesPythonReference` in `onnx_test.go`. It skips unless `SCHEMA_MCP_MODEL=1`. It uses the real `dir`, so the first run downloads about 220 MB. Against `testdata/reference.json`:
  - The token ids equal **exactly** for `query`, every `windows[].text`, every `probes[].text` and every `themes[].label`. Check ids first and stop on the first difference, printing both id lists.
    - The accent probe catches `strip_accents`.
    - The 600-word probe catches truncation: its ids are 512 long.
  - `Scorer{Enc: o}.Score(keyword, page.txt, labels)` is within `|Δ| ≤ 2e-5` of each theme's `keyword` and `page` values.
  - Each label's decision against 0.40 and against 0.55 equals the reference's. Freight transport's keyword score of 0.400098 must pass.
- [ ] **Step 6: Implement `onnx.go`.**
  - Load once, under `o.mu`, on the first `Encode`. A failed load returns its error and is retried on the next call.
  - `ensure` the assets, then call `ort.SetSharedLibraryPath` and `ort.InitializeEnvironment` (process-wide, guarded by a package `sync.Once` that keeps its error), then `ort.NewDynamicAdvancedSession(model, inputs, outputs, nil)`.
  - Load the tokenizer with `pretrained.FromFile(tokenizer.json)` and set `WithTruncation(&tokenizer.TruncationParams{MaxLength: 512, Strategy: tokenizer.LongestFirst})`.
  - Then call `WithNormalizer(normalizer.NewBertNormalizer(true, true, true, true))` (clean text, lowercase, Chinese chars, strip accents). sugarme reads `"strip_accents": null` as false, but the Hugging Face meaning is "follow lowercase", which is true.
  - Encode with `EncodeSingle(text, true)` and use `Ids`, `AttentionMask` and `TypeIds`.
  - On platforms other than linux/amd64, `Encode` returns `no embedding runtime download for <GOOS>/<GOARCH>`.
  - `onnx_nocgo.go` defines the same API. Its `Encode` returns `schema-mcp was built without cgo, so the embedding model cannot run`.
- [ ] **Step 7: Run the parity test. This step is required.** Run `SCHEMA_MCP_MODEL=1 go test ./internal/embed/ -run ONNX -v -timeout 20m`. Expected: PASS.
  - If the ids differ in a way the normalizer fix doesn't explain, replace sugarme with `github.com/daulet/tokenizers` (Hugging Face's Rust tokenizer, cgo) and re-run.
  - If the ids match but the cosines don't, compare one label's vector with Python's, element by element. Check `token_type_ids` (all 0), the attention mask, CLS pooling and float64 normalisation, in that order. Batching was ruled out on 2026-10-25.
- [ ] **Step 8: Run `go test ./internal/embed/` and `CGO_ENABLED=0 go build ./...`.** Expected: PASS, and the build succeeds.
- [ ] **Step 9: Commit** with the message `feat(embed): run bge-base-en-v1.5 locally with pinned downloads`.

---

### Task 7: Page links, the shared TextRazor call and `Progress`

**Files:**
- Modify: `internal/research/pages.go` (`pageJS`, `rawPage`, `cachedPage`, `fetchPage`), `internal/research/linking.go`, `internal/research/google.go`, `internal/research/serp.go` (`Researcher`)
- Test: `internal/research/pages_test.go`, `internal/research/linking_test.go`, `internal/research/live_test.go`

**Interfaces:**
- Consumes `azurelang.Result` (Task 4).
- Produces:
  - `type pageLink struct { Href string; Text string }` with the JSON tags `href` and `text`.
  - `rawPage.Links []pageLink` (`links`) and `cachedPage.Links []pageLink` (`links,omitempty`).
  - `func (r *Researcher) analyze(ctx context.Context, text string) (res textrazor.Result, cached bool, err error)`. It caches by SHA-256 in bucket `textrazor` with `pageTTL`.
  - `func Progress(ctx context.Context, msg string)`.
  - `type BingLinker interface { Link(ctx context.Context, text, language string) (azurelang.Result, error) }`.
  - `type ThemeScorer interface { Model() string; Score(ctx context.Context, keyword, page string, labels []string) (kw, pg []float64, err error) }`.
  - New fields `Researcher.Bing BingLinker` and `Researcher.Themes ThemeScorer`. When either is nil, select_entities skips that part.

- [ ] **Step 1: Write the failing tests**
  - `TestFetchPageKeepsMainLinks`: `pageFake` returns a `rawPage` with two links, and `fetchPage`'s result has the same `Links`.
  - `TestAnalyzeCachesByTextHash`: the second call with the same text gives `cached == true` and the fake linker is called once.
  - `TestLivePageJSReadsMainLinks` in `live_test.go` (`SCHEMA_MCP_LIVE=1`):
    - `httptest.NewServer` serves `testdata/select/page.html` at `/moving-cost-calculator/`;
    - real ChromiumFish runs `fetchPage`;
    - `Links` holds absolute hrefs on the test server's host ending `/cost-to-move-a-pool-table/`, `/cost-to-move-a-piano/` and `/cost-to-move-a-hot-tub/`, and the pool-table link's text contains `Pool table`;
    - `Links` holds no href ending `/privacy-policy/`, `/about-us/` or `/terms-and-conditions/`. Those links are outside `<main>`, in hidden chrome.
- [ ] **Step 2: Run `go test ./internal/research/ -run 'FetchPageKeeps|AnalyzeCaches'`.** Expected: FAIL.
- [ ] **Step 3: Implement.**
  - `pageJS` adds `links: [...root.querySelectorAll('a[href]')].filter(a => a.getClientRects().length > 0).map(a => ({href: a.href, text: (a.innerText || '').trim().replace(/\s+/g, ' ')})).filter(l => l.text).slice(0, 500)` to its result, computed after the chrome is hidden.
    - The `getClientRects` filter is what drops hidden links. `innerText` on an element that isn't rendered returns its `textContent`, so empty text doesn't mark a link as hidden.
    - Don't filter with `!a.closest(chrome)`. `hide()` deliberately leaves visible any chrome that wraps the root, and that filter would drop every link on form-wrapped pages.
  - `linkEntities` calls `analyze`. `Progress` wraps `progress`.
- [ ] **Step 4: Run the tests.** Run `go test ./internal/research/`, then `SCHEMA_MCP_LIVE=1 go test ./internal/research/ -run LivePageJS -v`. Expected: PASS for both, and every existing `page_terms` test is still green.
- [ ] **Step 5: Commit** with the message `feat(research): read main-content links and share the TextRazor call`.

---

### Task 8: Keyword words, focus and exact-title lookups

**Files:**
- Create: `internal/research/selection_keywords.go`, `internal/research/selection_keywords_test.go`
- Modify: `internal/research/entities_test.go` (`fakeWiki`)

**Interfaces:**
- Consumes `resolver`, `fetch`, `wikiAPI`, `titles` and `ptrs` from `entities.go`.
- Produces:
  - `type FocusWord struct { InTitle bool; InH1 bool }`, with the JSON tags `in_title` and `in_h1`.
  - `type KeywordLookup struct { Phrase, Result, Title string; Chosen []string; Links int }`, with the JSON tags `phrase`, `result`, `title,omitempty`, `chosen,omitempty` and `links,omitempty`.
    - `Result` is `exact`, `missing`, `disambiguation` or `redirect`.
    - `Links` is the top link count.
  - `type titleQID struct { Title, QID string }`.
  - `type lookupHit struct { Phrase, Title, QID, How string }`. `How` is `exact`, `disambiguation` or `redirect`.
  - `func contentWords(keyword string) []string`: lowercased, split on anything that isn't a letter or digit, stop words dropped, deduplicated, in order.
  - `func stem(w string) string`: the spike's rule. Strip the first of `ings`, `ing`, `ers`, `er`, `es`, `s` that leaves at least 3 letters.
  - `func sharesStem(title string, words []string) bool`.
  - `func keywordGrams(keyword string) []string`: every run of adjacent words, by start index and then length. Drop a run that starts or ends with a stop word, and drop a single stop word.
  - `func focus(words []string, title, h1 string) map[string]FocusWord`.
  - `func sameTitle(a, b string) bool`: underscores become spaces, the first rune is case-folded, and the rest must match exactly.
  - `func (r *resolver) keywordLookups(grams []string, known []titleQID) ([]KeywordLookup, []lookupHit, error)`.
  - `func (r *resolver) linkCounts(options, known []string) (map[string]int, error)`.
- Stop words (English only; my call, Review Focus 2): `a about an and are as at be by does for from how i in is me my near of on or the to vs what with you your`.
  - `it`, `can` and `do` are left out of the list on purpose, because they are nouns in real keywords such as "IT support" and "trash can cleaning".

**Lookup algorithm:**
1. Look up every gram as a title, first letter upper-cased, through `r.titles`, with titles sorted inside each batch of 50.
2. Add every **exact** hit first. An exact hit has a QID and isn't already in `known` or among earlier hits.
3. Then settle each disambiguation and redirect, in gram order, against `known` plus the exact hits.
   - A disambiguation page's options come from the `generator=links` query `resolve` already uses, dropping missing and disambiguation pages.
   - A redirect has one option: its target.
4. `linkCounts` counts, for each option:
   - its links to known titles: `prop=links`, `titles=<options batch of 50>`, `pltitles=<known batch of 50>`, `pllimit=max`, `redirects=1`;
   - plus the known titles' links to it: the same query reversed.
   Follow each response's `continue` (all its keys) until there is none.
5. Choose the options with the highest count, when that count is at least 1, sorted by title. A redirect needs a count of at least 1.

- [ ] **Step 1: Extend `fakeWiki`.**
  - Add `articleLinks map[string][]string`, which answers `prop=links` restricted to `pltitles`.
  - Add `linksPerResponse int`. When it is above 0, answers are cut there with `"continue":{"plcontinue":"<n>","continue":"||"}` and resumed when `plcontinue` is sent.
  - Add `rawItems map[string]json.RawMessage`, returned as is by `wbgetentities`.
  - Add `wikiItem.sitelink`, emitted as `"sitelinks":{"enwiki":{"site":"enwiki","title":…}}`.
- [ ] **Step 2: Write the failing tests**
  - `TestContentWordsDropsStopWords`: `"Cost to move a Piano"` gives `["cost","move","piano"]`, `"movers near me"` gives `["movers"]`, and `"IT support"` gives `["it","support"]`.
  - `TestKeywordGramsSkipStopWordEdges`: `"cost to move a piano"` gives `cost`, `cost to move`, `cost to move a piano`, `move`, `move a piano`, `piano`, in that order.
  - `TestKeywordGramsAllRuns`: `"moving cost calculator"` gives `moving`, `moving cost`, `moving cost calculator`, `cost`, `cost calculator`, `calculator`.
  - `TestStem`: moving→mov, movers→mov, calculator→calculat, costs→cost, gas→gas.
  - `TestSharesStem`: "Moving company" with `["moving"]` is true; "Truck" with `["moving","cost","calculator"]` is false.
  - `TestFocusMatchesStems`: the title "Moving Cost Calculator: Free Estimate" and the H1 "Movers' price guide" give moving `{true,true}`, cost `{true,false}` and calculator `{true,false}`.
  - `TestSameTitle`: `Moving_company`/`moving company` is true, `ZIP code`/`ZIP Code` is false, and `Piano`/`piano` is true.
  - `TestKeywordLookupsExactAndMissing`: `cost` gives `exact` with the hit Cost; `moving cost` gives `missing` and no hit.
  - `TestKeywordLookupsDisambiguationTieKeepsBoth`: "Moving" lists A, B and C. A and B each have 2 links to or from the known titles and C has 0, so the chosen options are `[A, B]` (sorted) with `Links: 2`.
  - `TestKeywordLookupsDisambiguationWithoutLinksAddsNothing`: chosen is empty and there are no hits.
  - `TestKeywordLookupsRedirectNeedsALink`: a redirect whose target doesn't link to or from a known title gives `Result: "redirect"` and no hit; with one link, there is one hit with `How: "redirect"`.
  - `TestKeywordLookupsSkipKnownQIDs`: an exact hit whose QID is already in `known` adds no hit.
  - `TestLinkCountsFollowsContinue`: with `linksPerResponse: 1` and 3 links, the count is 3.
  - `TestLinkCountsBatchesFiftyTitles`: 120 options give 3 batches, and every option is counted.
  - `TestKeywordLookupsNeverSearch`: the fake fails the test on any `list=search`, `srsearch` or `wbsearchentities` call.
- [ ] **Step 3: Run `go test ./internal/research/ -run 'ContentWords|KeywordGrams|Stem|Focus|SameTitle|KeywordLookups|LinkCounts'`.** Expected: FAIL.
- [ ] **Step 4: Implement `selection_keywords.go`.**
- [ ] **Step 5: Run the same command.** Expected: PASS, along with all the existing `research` tests.
- [ ] **Step 6: Commit** with the message `feat(research): look keyword words up as exact Wikipedia titles`.

---

### Task 9: Identity by QID, the entity node and Bing IDs

**Files:**
- Create: `internal/research/selection_identity.go`, `internal/research/selection_identity_test.go`
- Modify: `internal/research/entities.go` (`wdEntity` gets `Sitelinks`)

**Interfaces:**
- Consumes `sameTitle` (Task 8), `schema.Entity.KGMIDs`, `schema.WikidataID` and `schema.BingEntityID` (Task 2), and `azurelang.Entity` (Task 4).
- Produces:
  - `wdEntity.Sitelinks map[string]struct{ Title string }` (`sitelinks`).
  - `func (e wdEntity) ranked(prop string) []string`: like `values`, but with preferred-rank values first and claim order otherwise.
  - `func (r *resolver) articleItems(qids []string) (map[string]wdEntity, error)`: `wbgetentities`, `props=sitelinks|claims`, `sitefilter=<lang>wiki`, with QIDs deduplicated, sorted and batched 50 at a time.
  - `type identity struct { Title, KGMID, GoogleKGID, P9885 string; KGMIDChecked bool; Classes []string }`.
  - `func checkIdentity(it wdEntity, found bool, lang, title, freebase string) (identity, string)`.
    - `title` is "" for a theme. The returned reason is "" when the identity is valid.
  - `func entityThing(lang string, s Selected, classes []string) json.RawMessage`.
  - `func attachBing(sel []*Selected, p9885 map[string]string, ents []azurelang.Entity, lang string) []string`.
    - Keys of `p9885` are QIDs. It returns notes.
  - `const ReasonIdentityMismatch = "identity_mismatch"` and `ReasonNotAnArticle = "not_an_article"`.

**checkIdentity rules**, applied in this order:
1. Not found, or no sitelink for `<lang>wiki`: `identity_mismatch`.
2. `title != ""` and `!sameTitle(sitelink, title)`: `identity_mismatch`.
3. P31 contains Q4167410, Q4167836 or Q13406463: `not_an_article`.
4. `freebase != ""`, P646 is present and `freebase` isn't among its values: `identity_mismatch`.
5. Otherwise valid:
   - `KGMID` is the first ranked P646 and `GoogleKGID` the first ranked P2671;
   - `KGMIDChecked` is true when `freebase` equals that P646;
   - `Classes` holds at most 2 entries: P31 values first, filled with P279 values, without repeats.

**entityThing**, built with `schema.Entity{Name, Wikidata: QID, KGMID, KGMIDs: [GoogleKGID]}`. For `en`, add `Wikipedia` and `DBpedia` from the name; otherwise add `SameAs: [Wikipedia URL in lang]`. Then set on the `*Thing`:
- `Identifier`: `Vals(WikidataID(qid), GoogleMachineID(m)…, BingEntityID(b))`, with absent IDs left out;
- `AlternateName`: `s.Matched` without any value equal to the name ignoring case, omitted when empty;
- `AdditionalType`: `schema.Wikidata(c)` for each class, omitted when empty.

**attachBing:**
- An Azure entity matches a selected entity when its `url` host is `<lang>.wikipedia.org`, and its unescaped `/wiki/` title is `sameTitle` with the selected entity's name.
- The match sets `BingID`, unless the item's P9885 differs from Azure's `bingId`. In that case the entity gets no Bing ID and a note names it.
- A match that agrees with P9885 sets `BingChecked`.

- [ ] **Step 1: Write the failing tests**
  - `TestCheckIdentityPasses`: the sitelink is "Moving company" for the title `Moving_company`, P646 is `/m/0bwxkd` and the freebase is `/m/0bwxkd`, giving `KGMIDChecked` true.
  - `TestCheckIdentityTitleMismatch`: the sitelink "Stairwell" for the title "Stairs" gives `identity_mismatch`.
  - `TestCheckIdentityGoogleIDMismatch`: P646 `/m/1` against freebase `/m/2` gives `identity_mismatch`.
  - `TestCheckIdentityUncheckedWhenOneSideLacksTheID`: no freebase, or no P646, keeps the entity with `KGMIDChecked` false. With no P646, `KGMID` is "".
  - `TestCheckIdentityNoBorrowedGoogleID`: an item with P694 pointing to an item that holds P646 still gives `KGMID` "".
  - `TestCheckIdentityRejectsNonArticles`: one case each for Q4167410, Q4167836 and Q13406463 gives `not_an_article`.
  - `TestCheckIdentityNoSitelink` gives `identity_mismatch`.
  - `TestCheckIdentityThemeTakesTheSitelinkTitle`: `title ""` with the sitelink "Office supplies" gives `Title` "Office supplies".
  - `TestCheckIdentityClassesStopAtTwoPreferredFirst`: P31 is `[Q1 normal, Q2 preferred]` and P279 is `[Q3]`, giving `[Q2, Q1]`; P31 `[Q1]` with P279 `[Q3, Q4]` gives `[Q1, Q3]`.
  - `TestEntityThingSameAsIsExactlyTheFiveFamilies`: for `en`, the sameAs list is exactly Wikipedia, Wikidata, `?kgmid=/m/…`, `?kgmid=/g/…` and DBpedia, in that order. For `de` it is exactly `https://de.wikipedia.org/wiki/…`, Wikidata and the kgmid links, with no DBpedia.
  - `TestEntityThingIdentifiers`: the identifiers are Wikidata, GoogleMachineID `/m/` and `/g/`, and BingEntityID, in order, with no `@id`.
  - `TestEntityThingAlternateNameLeavesOutTheName`: matched `["pool tables","Pool table","billiard table"]` for the name "Billiard table" gives `["pool tables","Pool table"]`.
  - `TestAttachBingExactTitleOnly`: the Azure URL `…/wiki/Piano` attaches; `…/wiki/Pianos` and `https://es.wikipedia.org/wiki/Piano` for `en` don't.
  - `TestAttachBingP9885Disagreement`: the Bing ID is left out, the entity is kept, and the note names it. On agreement `BingChecked` is true.
- [ ] **Step 2: Run `go test ./internal/research/ -run 'CheckIdentity|EntityThing|AttachBing'`.** Expected: FAIL.
- [ ] **Step 3: Implement `selection_identity.go`.** Also create `selection.go` now, holding only the `Selected` type with the fields Task 10 lists, so this task compiles. Task 10 fills in the rest of the file.
- [ ] **Step 4: Run the same command.** Expected: PASS.
- [ ] **Step 5: Commit** with the message `feat(research): check entity identity by QID and build lean Thing nodes`.

---

### Task 10: `SelectEntities`

**Files:**
- Create: `internal/research/selection.go`, `internal/research/selection_test.go`

**Interfaces:**
- Consumes everything from Tasks 7–9.
- Produces:
  - `type SelectSettings struct { MinRelevance, MinTopic, MinKeywordSimilarity, MinPageSimilarity float64 }`, with the JSON tags `min_relevance`, `min_topic`, `min_keyword_similarity` and `min_page_similarity`.
  - `var DefaultSelectSettings = SelectSettings{0.3, 0.5, 0.40, 0.55}`.
  - `type SelectQuery struct { URL, Keyword, Language string; Settings SelectSettings }`.
  - `func (r *Researcher) SelectEntities(ctx context.Context, q SelectQuery) (SelectReport, error)`.
  - The constants `SourcePage = "page"`, `SourceKeyword = "keyword"`, `SourceTheme = "theme"`, `ReasonBelowRelevance = "below_relevance"`, `ReasonBelowTopic = "below_topic"` and `ReasonBelowSimilarity = "below_similarity"`.
  - `SelectReport`, `Selected`, `Dropped`, `TextRazorUsage{Units int; Cached bool}` and `AzureUsage{Records int; Cached bool}`. Their field names and JSON tags are exactly the spec's **Output** block.
    - Selected numbers that may be absent (`relevance`, `topic`, `confidence`, `keyword_similarity`, `page_similarity`) are `*float64` with `omitempty`, so a topic score of 0 still shows.
    - `qualified_by` is `[]string`.
    - `thing` is `json.RawMessage`.

**The rules the code follows. Each one cites a spec section.**
- **Validation:**
  - the URL must be `http` or `https`;
  - the keyword must not be blank;
  - every setting must be in [0, 1];
  - `r.Linker == nil` is the error `errors.New(noLinkerNote)`;
  - main text that is empty after trimming gives `no main text on <url>`.
- **Language** is `q.Language`, else the page's `lang`, lowercased, before `-` or `_`. If that is invalid against `languageCode`, the language is `en`.
- **The page** is read with `fetchPage` with no cache, so an edited page is always read fresh. `words` is `len(strings.Fields(text))`.
- **The TextRazor text** is `textrazor.Clip(strings.TrimSpace(title+"\n\n"+text), max)`. It is also the page text given to `Themes.Score`. A clip adds the note `text cut to <n> KB (textrazor.max_kb_per_page)`.
- **Grouping:** follow spec step 4, keyed by `wikidataId`, with the topic score being the highest score among topics sharing that QID.
- **Own-site links:**
  - same host as `final_url`, ignoring case and a leading `www.`;
  - not the page itself (same path, ignoring a trailing `/` and the fragment);
  - not the site root;
  - the text contains a matched form as whole words, ignoring case;
  - the hrefs are deduplicated and sorted.
- **Gate:** a page entity qualifies by scores when relevance ≥ min **and** topic ≥ min, or by having a link. A page entity that fails is dropped with `below_relevance` when its relevance is under the minimum, otherwise with `below_topic`.
- **Keyword page entities** are qualifying ones that pass the scores and whose title `sharesStem` with `contentWords`.
- **Lookups** come from Task 8, with `known` being the keyword page entities.
- **Each QID is selected at most once:**
  - a lookup hit on any page entity's QID moves that entity to `about` with `source: keyword`, keeping its page measurements;
  - a selected QID never appears in `dropped`.
- **Themes** are topics with a QID that isn't a page entity (qualifying or not) or a lookup hit.
  - Run `checkIdentity` with `title ""` first, and drop failures with its reason.
  - Score only the articles. Compare the **unrounded** cosines with `>=`, and report them rounded to 3 decimals.
  - With `r.Themes == nil`, or a `Score` error, there are no themes and the note says `no themes: <why>`.
  - `model` is `r.Themes.Model()`.
- **Bing:**
  - when `r.Bing == nil`, the note says `no Bing IDs: set azure_language.endpoint and azure_language.api_key in the schema-mcp config file`;
  - when `!azurelang.Supported(lang)`, the note says `no Bing IDs: Azure entity linking reads only English and Spanish`;
  - otherwise the result is cached in bucket `azure` under the key `sha256(lang+"\n"+text)` with `pageTTL`, and passed to `attachBing`.
- **Focus note:** for each content word missing from the title or the H1 (the first `h1:` heading), the note says `the title lacks "<w>"` or `the H1 lacks "<w>"`, followed by `: the page's themes may describe its own topic rather than the keyword's`.
- **Order and qualified_by:** follow spec step 9 and the Output block. Ties on relevance break by topic, then mentions, then name. Ties on topic score break by name. Notes are joined with `; `.

- [ ] **Step 1: Write the failing tests.** Use a combined fake page: `pageJS` returns a given `rawPage`, and everything else goes to `fakeWiki.page()`. Add a `fakeScorer` (label → two cosines) and a `fakeBing`.
  - `TestSelectEntitiesGateAndRoles`:
    - A has rel 0.8 and topic 1, with title "Moving company": it goes to `about` with qualified_by `[scores, keyword]`.
    - B has rel 0.6 and topic 0.9: it goes to `mentions` as `[scores]`.
    - C has rel 0.1 and topic 0, with a link: `mentions` `[link]`, last.
    - D has rel 0.2: dropped as `below_relevance`.
    - E has rel 0.4 and topic 0.1: dropped as `below_topic`.
  - `TestSelectEntitiesLinkToAnotherHostDoesNotCount`: C's link to `https://other.example/x` leaves C dropped.
  - `TestSelectEntitiesOwnHostForms`: the page reached through the redirect `http://a.example/p` to `https://www.a.example/p/` counts the link `https://a.example/q/` and ignores `https://www.a.example/p/#top` and `https://a.example/`. This is Review Focus 4.
  - `TestSelectEntitiesThemesNeedBothCutoffs`:
    - theme T1 (0.45, 0.60) goes in `about` with `source: theme` and `[similarity]`;
    - T2 (0.39, 0.60) and T3 (0.45, 0.54) are dropped as `below_similarity`, with both similarities;
    - T4 (0.400098, 0.60) is kept, which shows unrounded `>=`.
  - `TestSelectEntitiesThemeNameIsTheSitelink`: the label "Office equipment" with the sitelink "Office supplies" gives the name "Office supplies".
  - `TestSelectEntitiesCategoryThemeDropped`: P31 Q4167836 gives `not_an_article`.
  - `TestSelectEntitiesWithoutScorerHasNoThemes`: no themes, and the note contains `no themes`.
  - `TestSelectEntitiesWithoutLinkerFails`: the error contains `textrazor.api_key`.
  - `TestSelectEntitiesFocusNote`: the title lacks "calculator", and the note contains `the title lacks "calculator"`.
  - `TestSelectEntitiesWithoutBingNotes`: the note contains `azure_language.endpoint`, and no entity has `bing_id`.
  - `TestSelectEntitiesBingNotForGerman`: a page with `lang` `de` makes no `fakeBing` call, the note names English and Spanish, and the things have no DBpedia. This is Review Focus 3.
  - `TestSelectEntitiesLanguageFromPage`: `en-US` gives `language: en`, and `""` gives `en`.
  - `TestSelectEntitiesLookupHitMovesAPageEntity`: a page entity failing the gate, whose QID is a lookup hit, appears once in `about` with `source: keyword` and not in `dropped`.
  - `TestSelectEntitiesRejectsAPageWithNoText`.
  - `TestSelectEntitiesUsesTheTextRazorCache`: the second run reports `textrazor.cached: true`, and the linker is called once.
- [ ] **Step 2: Run `go test ./internal/research/ -run SelectEntities`.** Expected: FAIL.
- [ ] **Step 3: Implement `selection.go`.** Use one `Browser.Use` for the page read, then one for all Wikipedia and Wikidata work (`resolver.open`, lookups, `articleItems` over every QID at once).
- [ ] **Step 4: Run `go test ./internal/research/`.** Expected: PASS.
- [ ] **Step 5: Commit** with the message `feat(research): add SelectEntities`.

---

### Task 11: The golden calibration test

**Files:**
- Create: `internal/research/selection_golden_test.go`, `internal/research/selection_record_test.go`, `internal/research/testdata/select/wiki.json`

**Interfaces:**
- Consumes `SelectEntities` (Task 10), the `fakeWiki` extensions (Task 8) and the Task 1 fixtures.
- Produces:
  - `func calibrationPage(t *testing.T, base string) rawPage`, built from `page.html` and `main.txt`:
    - the `<title>` and `<html lang>`;
    - `Headings: ["h1: <first h1 text>"]`;
    - `Text`: `main.txt` without its first line, which is the title;
    - `Links`: every `<a href>` inside `<main>…</main>`, with the href resolved against `base` and the text stripped of tags, HTML-unescaped and whitespace-collapsed.
  - `func loadWikiFixture(t *testing.T) *fakeWiki`.
  - The `wiki.json` shape is `{"items": {qid: raw entity}, "pages": {title: {"qid", "disambig", "missing"}}, "redirects": {from: to}, "options": {disambiguation title: [titles]}, "links": {title: [titles]}}`, with sorted keys.

- [ ] **Step 1: Write `TestSelectEntitiesCalibration`.** It uses the fake page (`calibrationPage(t, "http://localhost:8080/moving-cost-calculator/")` plus `loadWikiFixture`), the fake linker (`textrazor.json`'s `response.entities` and `response.topics` for any text), and the fake scorer (`cosines.json`; a label that is missing fails the test). It asserts:
  - `about` names, in order: Calculator, Moving company, Cost, Relocation (personal), Relocation service, Transport, Freight transport, Business economics, Pricing, Land transport, Accounting.
  - `mentions` names, in order: Truck, Stairs, ZIP Code, Federal Motor Carrier Safety Administration, Weight, Piano, Hot tub, Billiard table.
    - Hot tub's `qualified_by` contains `scores`.
    - Billiard table's is `[link]`, and its `linked_to` is `["http://localhost:8080/cost-to-move-a-pool-table/"]`.
  - In `dropped`, Email is `below_relevance`, and Computing is `below_similarity` with keyword 0.409 and page 0.492.
  - Every `source: page` entity has `kgmid_checked: true`; every keyword and theme entity has `false`.
  - The `keyword_lookups` results:
    - moving is `disambiguation`, chosen `[Relocation (personal), Relocation service]`, links 2;
    - cost and calculator are `exact`;
    - moving cost, cost calculator and moving cost calculator are `missing`.
  - `words == len(strings.Fields(calibrationPage(...).Text))`, `language` is `en`, and `title` is the page's `<title>`.
  - Billiard table's `thing`, unmarshalled into a map:
    - `name` "Billiard table" and `@type` "Thing", with no `@id`;
    - `alternateName` `["pool tables","Pool table"]`;
    - sameAs exactly `[https://en.wikipedia.org/wiki/Billiard_table, https://www.wikidata.org/wiki/Q3500725, https://www.google.com/search?kgmid=/m/04p0qw, http://dbpedia.org/resource/Billiard_table]`;
    - identifiers `Wikidata Q3500725` and `GoogleMachineID kg:/m/04p0qw`;
    - `additionalType` holding at most 2 `https://www.wikidata.org/wiki/Q…` URLs. They were Q768186 and Q14748 at calibration; assert the values from the recording.
  - The test skips with `record wiki.json first: SCHEMA_MCP_RECORD=1 go test ./internal/research -run RecordSelect` while `wiki.json` is missing.
- [ ] **Step 2: Write `TestRecordSelectFixtures`** (`SCHEMA_MCP_RECORD=1`). It runs `SelectEntities` with the same fakes, except that the browser is real ChromiumFish (`browser.New("go-json-schema-record")`) wrapped by a recorder. The recorder:
  - answers `Navigate` to the page URL and the `pageJS` eval from `calibrationPage`, without loading the page;
  - passes every other call through;
  - takes each `fetchJS` call's URLs and responses apart into `wiki.json`'s maps, per QID and per title (normalised, redirect, page, options, links), never keyed by the raw URL;
  - writes `wiki.json` with sorted keys and two-space indentation.
- [ ] **Step 3: Run the golden test.** Run `go test ./internal/research/ -run SelectEntitiesCalibration -v`. Expected: SKIP with the record message.
- [ ] **Step 4: Record. This step is required.** Run `SCHEMA_MCP_RECORD=1 go test ./internal/research/ -run RecordSelect -v -timeout 10m`. Expected: PASS, and `wiki.json` is written. Its `items` hold every page QID, both Relocation items and the theme QIDs.
- [ ] **Step 5: Run `go test ./internal/research/ -run SelectEntitiesCalibration -v`.** Expected: PASS. If it fails:
  - fix the code, or the compare rule;
  - re-record only if the fixture is wrong;
  - never edit the expected lists. The spec forbids it.
- [ ] **Step 6: Run the full suite.** Run `go test ./...` through `ctx_batch_execute`. Expected: every package passes.
- [ ] **Step 7: Commit** with the message `test(research): golden select_entities run on the calibration page`.

---

### Task 12: MCP tool, prompt, guide, README and wiring

**Files:**
- Modify: `internal/mcpserver/server.go`, `internal/mcpserver/server_test.go`, `internal/mcpserver/guide.md`, `cmd/schema-mcp/main.go`, `README.md`

**Interfaces:**
- Consumes `research.SelectEntities`, `research.DefaultSelectSettings`, `research.Progress`, `azurelang.Client`, `embed.Scorer` and `embed.NewONNX`.
- Produces:
  - the `select_entities` tool, with args `url`, `keyword`, `language` and `min_relevance`, `min_topic`, `min_keyword_similarity`, `min_page_similarity` as `*float64` with `omitempty`. A nil arg keeps the default.
  - `OutputSchema: outputSchema[research.SelectReport]()`.
  - The handler calls `r.SelectEntities(withProgress(ctx, req), …)`, as `serp` and `page_terms` do. Without that, the first-use model download reports no progress, and the client's idle timeout can end the call.

- [ ] **Step 1: Write the failing tests**
  - `TestServerListsAllTools` includes `select_entities`.
  - `TestSelectEntitiesToolWithoutTextRazorIsAToolError`: `Researcher{}` gives a tool error containing `textrazor.api_key`.
  - `TestSelectEntitiesToolRejectsACutoffAboveOne`: `min_topic: 2` gives a tool error.
  - `TestSelectEntitiesToolDescriptionSaysTheDefaultsAreJudgement`: the description contains `our judgement` and `one calibration page`.
  - `TestEntitySchemaPromptRequiresPageURL`: without `page_url`, the error says `page_url is required`. Making `page_url` required is my call: without our page there is no source for `about` and `mentions`.
  - `TestEntitySchemaPromptCallsSelectEntities`: the text contains `select_entities with url "http://localhost:8080/p/", keyword "plumber"` and `as given`, and doesn't contain `Choose 3-10 core entities`.
  - Update `TestEntitySchemaPromptEmbedsGuide`, `TestEntitySchemaPromptPassesCountryAndLanguage` and `TestEntitySchemaPromptUsesLinkedEntitiesAndBoldTerms` to pass `page_url`. Drop the `resolve_entities with language en` assertion and assert `select_entities` with `language en` instead.
  - `TestGuideEntityRows`: `guide` contains `select_entities` in the about and mentions rows and doesn't contain `about: 3–10`.
- [ ] **Step 2: Run `go test ./internal/mcpserver/`.** Expected: FAIL.
- [ ] **Step 3: Implement**
  - **The tool description must cover:**
    - our page is read, including `localhost`;
    - `about` is the keyword's own entities (from the page and from exact Wikipedia titles) plus the page's themes, which may not be named on the page;
    - `mentions` holds only entities the page names;
    - there are no caps;
    - each entry comes with its `thing` node to use as given;
    - the four defaults are "our judgement, set by eye from one calibration page, not from any Google source";
    - it needs a TextRazor key, while Azure (Bing IDs) and the local model (themes; about 220 MB downloaded on first use) are optional and their absence is explained in `note`.
  - **Instructions, workflow step 3:**
    > select_entities(page_url, keyword): the about and mentions entities for our page, each with a ready thing node. Use them as given: add none, drop none. page_terms entities and resolve_entities lookups no longer feed about or mentions; use resolve_entities for other entity properties such as areaServed.

    Also mention select_entities' outside calls (TextRazor, Azure) in the closing paragraph.
  - **The prompt:**
    - `page_url` is required;
    - step 3 becomes `Call select_entities with url %q, keyword %q and language <wiki>. Put its about and mentions in the graph as given.`;
    - step 4's "Choose 3-10…" line is removed.
  - **guide.md:**
    - The comment becomes `<!-- Adapted from SEO_FIELD_GUIDE in recipe-create-pipeline/scripts/lib/pipeline_common.py (2026-10-24). Update both together, except the about, mentions, alternateName and additionalType rules, which are schema-mcp's own since 2026-10-25. -->`.
    - Rows:
      - `about | Thing[] | The keyword's topic family from select_entities, as given: the keyword's own entities and the page's themes`;
      - `mentions | Thing[] | Entities the page names, from select_entities, as given`;
      - `alternateName | Text[] | On about/mentions entities: only words the page uses (select_entities sets them). Elsewhere: aliases, translations, synonyms and misspellings people search for`;
      - `additionalType | URL[] | On about/mentions entities: at most 2 Wikidata classes (select_entities sets them). Elsewhere: class URIs (Wikidata class, productontology, schema.org types, Wikipedia)`.
    - Volume targets: remove the about and mentions lines and add `- about, mentions: no targets; select_entities decides`. Mark the alternateName and additionalType lines `(not about/mentions entities)`.
    - Line 7: `use the ones select_entities or resolve_entities returns`.
  - **main.go:**
    - `res.Bing = &azurelang.Client{Endpoint: cfg.AzureLanguage.Endpoint, Key: cfg.AzureLanguage.APIKey}` when it is set;
    - when `defaultCacheDir() != ""`: `enc := embed.NewONNX(filepath.Join(defaultCacheDir(), "models"), research.Progress)`, `defer enc.Close()`, `res.Themes = &embed.Scorer{Enc: enc}`.
  - **README:**
    - a `select_entities(url, keyword)` row in the tools table;
    - a section "Entity selection (`select_entities`)" covering:
      - the rules in brief, with a link to the spec;
      - the defaults, as our judgement from one calibration page (with a link to the calibration data), still to be checked on a second page;
      - the IDs it emits;
      - the `azure_language` settings, including the 1 September 2028 retirement;
      - the model download (about 220 MB under the user cache dir, linux/amd64, needs cgo and gcc; without them, no themes);
      - the gated test commands (`SCHEMA_MCP_MODEL=1`, `SCHEMA_MCP_RECORD=1`, `SCHEMA_MCP_LIVE=1`).
- [ ] **Step 4: Run `go test ./...` and `CGO_ENABLED=0 go build ./...`.** Expected: PASS, and the build succeeds.
- [ ] **Step 5: Check end to end.** This is required when ChromiumFish and the TextRazor key are available.
  - Serve `internal/research/testdata/select/page.html` with `python3 -m http.server` from a scratch dir holding a copy, then `go install ./cmd/schema-mcp`.
  - Call `select_entities` through a stdio MCP client script (the venv has `mcp`), with that URL and `moving cost calculator`.
  - Expected:
    - the `about` and `mentions` names match Task 11's lists;
    - `model` names bge-base;
    - the note explains that no Bing IDs were added;
    - one TextRazor unit, or `cached: true`.
  - Report the result to the user, with any differences from the golden lists.
- [ ] **Step 6: Commit** with the message `feat(mcp): add select_entities tool and route about/mentions through it`.

---

## My calls (not the spec's or the owner's)

- **Embedding stack.** I chose onnxruntime_go v1.38.0 with ONNX Runtime 1.31.0, the same version google-seo-docs' Python uses, plus sugarme/tokenizer with its normalizer overridden. The downloads are pinned by SHA-256, and only linux/amd64 is supported.
- **Similarity cutoffs** compare the unrounded cosine and report it rounded to 3 decimals. The spike rounded before comparing. Only Safety (page 0.549987) would flip, and it fails the keyword cutoff anyway.
- **Stop words** are not looked up and not counted in the focus check (Review Focus 2).
- **Exact titles first:** every exact title is resolved before any disambiguation or redirect is settled, and tied options are sorted by title.
- **One selection per QID:** a lookup hit on a page entity's QID moves that entity to `about`.
- **Links:** self-links and root links don't count as own-site links, and `www.` matches the bare host.
- **No cache for our page.** Our own page is never served from the page cache, while TextRazor and Azure results are cached by the hash of the text.
- **The `entity_schema` prompt requires `page_url`.**
- **The page embedding** uses the exact text sent to TextRazor, which is title + blank line + main text.
- **Identifier names:** the `propertyID`s are `Wikidata`, `GoogleMachineID` and `BingEntityID`.
