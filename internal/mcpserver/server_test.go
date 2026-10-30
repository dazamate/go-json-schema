package mcpserver

import (
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dazamate/go-json-schema/internal/browser"
	"github.com/dazamate/go-json-schema/internal/cooldown"
	"github.com/dazamate/go-json-schema/internal/research"
	"github.com/dazamate/go-json-schema/internal/vocabtool"
)

// googlePage answers a Google results page with two results.
type googlePage struct{}

func (googlePage) Navigate(_ context.Context, url string) (string, error) { return url, nil }

func (googlePage) Eval(_ context.Context, _ string, out any) error {
	return json.Unmarshal([]byte(`{"results":[{"title":"A","href":"https://a.example/"},{"title":"B","href":"https://b.example/"}],"suggest":["x y"]}`), out)
}

type oneTab struct{}

func (oneTab) Use(_ context.Context, fn func(browser.Page) error) error { return fn(googlePage{}) }

func connect(t *testing.T, res *research.Researcher) *mcp.ClientSession {
	return connectWith(t, res, nil)
}

func connectWith(t *testing.T, res *research.Researcher, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	vocab, err := vocabtool.New()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := New(vocab, res, "test").Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, opts).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callText(t *testing.T, cs *mcp.ClientSession, tool string, args any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String(), res.IsError
}

func TestServerListsAllTools(t *testing.T) {
	cs := connect(t, &research.Researcher{Browser: oneTab{}})
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	slices.Sort(got)
	want := []string{"build_jsonld", "enum_values", "page_terms", "property_info", "resolve_entities", "select_entities", "serp", "type_properties"}
	if !slices.Equal(got, want) {
		t.Fatalf("tools = %v, want %v", got, want)
	}
}

func TestServerWithoutResearcherKeepsVocabTools(t *testing.T) {
	cs := connect(t, nil)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil || len(res.Tools) != 4 {
		t.Fatalf("tools = %d, %v", len(res.Tools), err)
	}
}

func TestBuildJSONLDKeepsDraftKeyOrder(t *testing.T) {
	cs := connect(t, nil)
	args := json.RawMessage(`{"draft":{"@type":"Recipe","name":"N","about":[{"@type":"Thing","name":"Lemon"}],"availability":"InStock"}}`)
	out, isErr := callText(t, cs, "build_jsonld", args)
	if isErr {
		t.Fatal(out)
	}
	var rep vocabtool.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rep.ScriptTag, `{"@context":"https://schema.org","@type":"Recipe","name":"N","about":[{"@type":"Thing","name":"Lemon"}]`) {
		t.Fatalf("script tag = %s", rep.ScriptTag)
	}
	if rep.Errors != 0 || len(rep.Issues) != 1 || rep.Issues[0].Severity != vocabtool.Info {
		t.Fatalf("issues = %+v (availability on Recipe should be info only)", rep.Issues)
	}
}

func TestBuildJSONLDAcceptsDraftAsString(t *testing.T) {
	cs := connect(t, nil)
	out, isErr := callText(t, cs, "build_jsonld", map[string]any{"draft": `{"@type":"Persn"}`})
	if isErr || !strings.Contains(out, `did you mean Person?`) {
		t.Fatalf("out = %s", out)
	}
}

func TestBuildJSONLDChecksEntitiesAgainstPageText(t *testing.T) {
	cs := connect(t, nil)
	draft := `{"@type":"WebPage","about":[{"@type":"Thing","name":"Lemon"},{"@type":"Thing","name":"Hajong marriage"}]}`
	out, isErr := callText(t, cs, "build_jsonld", map[string]any{"draft": draft, "page_text": "A lemon tart."})
	if isErr {
		t.Fatal(out)
	}
	var rep vocabtool.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Warnings != 1 || !strings.Contains(out, "Hajong marriage is not named on the page") {
		t.Fatalf("issues = %+v, want one warning for Hajong marriage", rep.Issues)
	}
}

func TestBuildJSONLDListsPageElements(t *testing.T) {
	cs := connect(t, nil)
	page := `<html><body><header><nav id="site-nav">Home</nav></header><main><p>Lemon tart</p></main><footer>ABN</footer></body></html>`
	draft := `{"@type":"WebPage","about":{"@type":"Thing","name":"Lemon"},"hasPart":{"@type":"SiteNavigationElement","cssSelector":"#site-nav"}}`
	out, isErr := callText(t, cs, "build_jsonld", map[string]any{"draft": draft, "page_html": page})
	if isErr {
		t.Fatal(out)
	}
	var rep vocabtool.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.PageElements) != 3 || rep.PageElements[1].MarkedBy != "$.hasPart" {
		t.Fatalf("page elements = %+v, want header, the marked nav and footer", rep.PageElements)
	}
	// header and footer unmarked; Lemon named by the page's text
	if rep.Warnings != 2 || !strings.Contains(out, "has no WPHeader") || !strings.Contains(out, "has no WPFooter") {
		t.Fatalf("issues = %+v", rep.Issues)
	}
}

func TestLookupErrorsAreToolErrors(t *testing.T) {
	cs := connect(t, nil)
	out, isErr := callText(t, cs, "type_properties", map[string]any{"type": "recipe"})
	if !isErr || !strings.Contains(out, "did you mean Recipe?") {
		t.Fatalf("out = %s, isErr = %v", out, isErr)
	}
}

func TestSerpToolReturnsResults(t *testing.T) {
	cs := connect(t, &research.Researcher{Browser: oneTab{}})
	out, isErr := callText(t, cs, "serp", map[string]any{"keyword": "x"})
	if isErr {
		t.Fatal(out)
	}
	var s research.SERP
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	if s.Engine != "google" || len(s.Results) != 2 || !slices.Equal(s.Autocomplete, []string{"x y"}) {
		t.Fatalf("serp = %+v", s)
	}
}

func TestEntitySchemaPromptEmbedsGuide(t *testing.T) {
	cs := connect(t, nil)
	res, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: "entity_schema", Arguments: map[string]string{"keyword": "lemon herb chicken", "page_type": "Recipe", "page_url": "http://localhost:8080/p/"}})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Messages[0].Content.(*mcp.TextContent).Text
	if !strings.Contains(text, `"lemon herb chicken"`) || !strings.Contains(text, "type_properties for Recipe") || !strings.Contains(text, "Volume targets") {
		t.Fatalf("prompt = %s", text)
	}
}

// wikiPage answers resolve_entities' in-page API fetches for "lemon".
type wikiPage struct{}

func (wikiPage) Navigate(_ context.Context, url string) (string, error) { return url, nil }

func (wikiPage) Eval(_ context.Context, expr string, out any) error {
	if expr == "location.origin" {
		return json.Unmarshal([]byte(`"https://en.wikipedia.org"`), out)
	}
	var urls []string
	if err := json.Unmarshal([]byte(strings.TrimSuffix(expr[strings.LastIndex(expr, "([")+1:], ")")), &urls); err != nil {
		return err
	}
	var res []map[string]any
	for _, u := range urls {
		body := `{"query":{"normalized":[{"from":"lemon","to":"Lemon"}],"pages":[{"title":"Lemon","pageprops":{"wikibase_item":"Q500"}}]}}`
		if strings.Contains(u, "wikidata") {
			body = `{"entities":{"Q500":{"id":"Q500","labels":{"en":{"value":"lemon"}},"claims":{"P646":[{"mainsnak":{"datavalue":{"value":"/m/09k_b"}}}]}}}}`
		}
		res = append(res, map[string]any{"status": 200, "body": body})
	}
	b, _ := json.Marshal(res)
	return json.Unmarshal(b, out)
}

type wikiTab struct{}

func (wikiTab) Use(_ context.Context, fn func(browser.Page) error) error { return fn(wikiPage{}) }

func TestResolveEntitiesToolReturnsThingNodes(t *testing.T) {
	cs := connect(t, &research.Researcher{Browser: wikiTab{}})
	out, isErr := callText(t, cs, "resolve_entities", map[string]any{"phrases": []string{"lemon"}})
	if isErr {
		t.Fatal(out)
	}
	var rep struct {
		Entities []struct {
			KGMID string         `json:"kgmid"`
			Thing map[string]any `json:"thing"`
		} `json:"entities"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatal(err)
	}
	if e := rep.Entities[0]; e.KGMID != "/m/09k_b" || e.Thing["name"] != "Lemon" {
		t.Fatalf("entity = %+v", e)
	}
}

// recordingTab is a Google results tab that records every URL it opens.
type recordingTab struct {
	mu   sync.Mutex
	urls []string
}

func (r *recordingTab) Use(_ context.Context, fn func(browser.Page) error) error {
	return fn(recordingPage{r})
}

type recordingPage struct{ tab *recordingTab }

func (p recordingPage) Navigate(ctx context.Context, u string) (string, error) {
	p.tab.mu.Lock()
	p.tab.urls = append(p.tab.urls, u)
	p.tab.mu.Unlock()
	return googlePage{}.Navigate(ctx, u)
}

func (p recordingPage) Eval(ctx context.Context, expr string, out any) error {
	return googlePage{}.Eval(ctx, expr, out)
}

func TestSerpToolPassesEveryGoogleParam(t *testing.T) {
	tab := &recordingTab{}
	cs := connect(t, &research.Researcher{Browser: tab})
	out, isErr := callText(t, cs, "serp", map[string]any{
		"keyword": "plumber", "country": "au", "language": "en-gb", "google_domain": "google.com.au", "page": 2,
		"time_range": "month", "safe_search": true, "results_language": "en", "results_country": "au",
		"no_autocorrect": true, "show_omitted": true, "web_only": true, "near": "Perth WA", "params": map[string]string{"tbm": "nws"},
	})
	if isErr {
		t.Fatal(out)
	}
	u, err := url.Parse(tab.urls[0])
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"gl": "au", "hl": "en-GB", "start": "10", "tbs": "qdr:m", "safe": "active", "lr": "lang_en",
		"cr": "countryAU", "nfpr": "1", "filter": "0", "udm": "14", "tbm": "nws", "near": "Perth WA"}
	for k, v := range want {
		if got := u.Query().Get(k); got != v {
			t.Errorf("%s = %q, want %q (url %s)", k, got, v, u)
		}
	}
	if u.Host != "www.google.com.au" {
		t.Errorf("host = %s", u.Host)
	}
	var s research.SERP
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	if s.SearchURL != tab.urls[0] || s.Results[0].Rank != 11 {
		t.Fatalf("serp = %+v", s)
	}
}

func TestSerpToolRejectsABadParam(t *testing.T) {
	cs := connect(t, &research.Researcher{Browser: &recordingTab{}})
	out, isErr := callText(t, cs, "serp", map[string]any{"keyword": "x", "time_range": "fortnight"})
	if !isErr || !strings.Contains(out, "time_range") {
		t.Fatalf("out = %s, isErr = %v", out, isErr)
	}
}

func TestSerpToolReportsTheCooldownAsProgress(t *testing.T) {
	var mu sync.Mutex
	var msgs []string
	opts := &mcp.ClientOptions{ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
		mu.Lock()
		msgs = append(msgs, req.Params.Message)
		mu.Unlock()
	}}
	cs := connectWith(t, &research.Researcher{Browser: &recordingTab{}, Google: cooldown.New(50*time.Millisecond, "")}, opts)
	for i, kw := range []string{"first", "second"} {
		params := &mcp.CallToolParams{Name: "serp", Arguments: map[string]any{"keyword": kw}}
		params.SetProgressToken(i + 1)
		if _, err := cs.CallTool(context.Background(), params); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(time.Second)
	for {
		mu.Lock()
		got := slices.Clone(msgs)
		mu.Unlock()
		if len(got) > 0 {
			if len(got) != 1 || !strings.Contains(got[0], "Google cooldown") {
				t.Fatalf("progress = %q, want one cooldown message", got)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no progress notification for the cooldown wait")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEntitySchemaPromptPassesCountryAndLanguage(t *testing.T) {
	cs := connect(t, nil)
	res, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: "entity_schema", Arguments: map[string]string{"keyword": "plumber", "page_url": "http://localhost:8080/p/", "country": "au", "language": "en-GB"}})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Messages[0].Content.(*mcp.TextContent).Text
	if !strings.Contains(text, `serp with keyword "plumber", country au and language en-GB`) {
		t.Fatalf("prompt = %s", text)
	}
	if step := selectStep(t, text); !strings.Contains(step, `select_entities with url "http://localhost:8080/p/", keyword "plumber" and language en.`) {
		t.Errorf("select_entities step = %s", step)
	}
}

// selectStep is the prompt's select_entities step.
func selectStep(t *testing.T, text string) string {
	t.Helper()
	i := strings.Index(text, "3. Call select_entities")
	if i < 0 {
		t.Fatalf("no select_entities step: %s", text)
	}
	step, _, _ := strings.Cut(text[i:], "\n")
	return step
}

func TestEntitySchemaPromptUsesLinkedEntitiesAndBoldTerms(t *testing.T) {
	cs := connect(t, nil)
	res, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: "entity_schema", Arguments: map[string]string{"keyword": "plumber", "page_url": "http://localhost:8080/p/"}})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Messages[0].Content.(*mcp.TextContent).Text
	for _, want := range []string{"bold_terms", "entities", "topics", "rho"} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt does not mention %s", want)
		}
	}
}

func TestSelectEntitiesToolWithoutTextRazorIsAToolError(t *testing.T) {
	cs := connect(t, &research.Researcher{})
	out, isErr := callText(t, cs, "select_entities", map[string]any{"url": "http://localhost:8080/p/", "keyword": "plumber"})
	if !isErr || !strings.Contains(out, "textrazor.api_key") {
		t.Fatalf("out = %s, isErr = %v", out, isErr)
	}
}

func TestSelectEntitiesToolRejectsACutoffAboveOne(t *testing.T) {
	cs := connect(t, &research.Researcher{})
	out, isErr := callText(t, cs, "select_entities", map[string]any{"url": "http://localhost:8080/p/", "keyword": "plumber", "min_topic": 2})
	if !isErr || !strings.Contains(out, "min_topic is 2: want 0 to 1") {
		t.Fatalf("out = %s, isErr = %v, want a tool error naming min_topic", out, isErr)
	}
}

// min_link_confidence is a TextRazor confidence, so any number from 0 up passes.
func TestSelectEntitiesToolLinkConfidenceCutoff(t *testing.T) {
	cs := connect(t, &research.Researcher{})
	out, isErr := callText(t, cs, "select_entities", map[string]any{"url": "http://localhost:8080/p/", "keyword": "plumber", "min_link_confidence": -1})
	if !isErr || !strings.Contains(out, "min_link_confidence is -1: want 0 or more") {
		t.Errorf("out = %s, isErr = %v, want a tool error naming min_link_confidence", out, isErr)
	}
	out, isErr = callText(t, cs, "select_entities", map[string]any{"url": "http://localhost:8080/p/", "keyword": "plumber", "min_link_confidence": 3})
	if !isErr || !strings.Contains(out, "textrazor.api_key") {
		t.Errorf("out = %s, isErr = %v, want min_link_confidence 3 to pass the check", out, isErr)
	}
}

// The graph settings are integers with their own ranges: a neighbour cap may
// be 0, a link count may not.
func TestSelectEntitiesToolGraphSettings(t *testing.T) {
	cs := connect(t, &research.Researcher{})
	for _, c := range []struct {
		arg  string
		val  int
		want string
	}{
		{"max_graph_neighbours", -1, "max_graph_neighbours is -1: want 0 or more"},
		{"min_graph_links", 0, "min_graph_links is 0: want 1 or more"},
	} {
		out, isErr := callText(t, cs, "select_entities", map[string]any{"url": "http://localhost:8080/p/", "keyword": "plumber", c.arg: c.val})
		if !isErr || !strings.Contains(out, c.want) {
			t.Errorf("%s: out = %s, isErr = %v, want a tool error containing %q", c.arg, out, isErr, c.want)
		}
	}
	// 0 passes for these two, so the call gets as far as the missing key.
	for _, arg := range []string{"min_mention_graph_links", "max_graph_neighbours"} {
		out, isErr := callText(t, cs, "select_entities", map[string]any{"url": "http://localhost:8080/p/", "keyword": "plumber", arg: 0})
		if !isErr || !strings.Contains(out, "textrazor.api_key") {
			t.Errorf("%s: out = %s, isErr = %v, want 0 to pass the check", arg, out, isErr)
		}
	}
}

func TestSelectEntitiesToolDescriptionSaysTheDefaultsAreJudgement(t *testing.T) {
	cs := connect(t, &research.Researcher{})
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "select_entities" {
			continue
		}
		for _, want := range []string{"our judgement", "two calibration pages"} {
			if !strings.Contains(tool.Description, want) {
				t.Errorf("description does not contain %q", want)
			}
		}
		return
	}
	t.Fatal("no select_entities tool")
}

func TestEntitySchemaPromptRequiresPageURL(t *testing.T) {
	cs := connect(t, nil)
	_, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: "entity_schema", Arguments: map[string]string{"keyword": "plumber"}})
	if err == nil || !strings.Contains(err.Error(), "page_url is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestEntitySchemaPromptDeclaresPageURL(t *testing.T) {
	cs := connect(t, nil)
	res, err := cs.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Prompts {
		if p.Name != "entity_schema" {
			continue
		}
		for _, a := range p.Arguments {
			if a.Name == "page_url" {
				if !a.Required {
					t.Errorf("page_url is not required: %+v", a)
				}
				return
			}
		}
		t.Fatalf("entity_schema does not declare page_url: %+v", p.Arguments)
	}
	t.Fatal("no entity_schema prompt")
}

func TestEntitySchemaPromptCallsSelectEntities(t *testing.T) {
	cs := connect(t, nil)
	res, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: "entity_schema", Arguments: map[string]string{"keyword": "plumber", "page_url": "http://localhost:8080/p/"}})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Messages[0].Content.(*mcp.TextContent).Text
	// With no language given, select_entities uses the page's lang.
	if step := selectStep(t, text); !strings.Contains(step, `select_entities with url "http://localhost:8080/p/" and keyword "plumber".`) ||
		strings.Contains(step, "language") || !strings.Contains(step, "as given") {
		t.Fatalf("select_entities step = %s", step)
	}
	if strings.Contains(text, "Choose 3-10 core entities") {
		t.Fatalf("prompt still has the old about/mentions step: %s", text)
	}
}

func TestGuideEntityRows(t *testing.T) {
	for _, row := range []string{"| about |", "| mentions |"} {
		i := strings.Index(guide, row)
		if i < 0 {
			t.Fatalf("guide has no %s row", row)
		}
		line, _, _ := strings.Cut(guide[i:], "\n")
		if !strings.Contains(line, "select_entities") {
			t.Errorf("row %q does not mention select_entities", line)
		}
	}
	if strings.Contains(guide, "about: 3–10") {
		t.Error("guide still has the about volume target")
	}
}

// TestPageGraphGuidance keeps the three places that tell an agent how to
// shape the graph in step: the guide's Page graph section, the server
// instructions and the entity_schema prompt.
func TestPageGraphGuidance(t *testing.T) {
	i := strings.Index(guide, "## Page graph")
	if i < 0 {
		t.Fatal("guide has no Page graph section")
	}
	section := guide[i:]
	for _, want := range []string{"hasPart", "isPartOf", "mainEntityOfPage", "Brand", "potentialAction", "DesktopWebPlatform", "{\"@id\": ...}", "stays a Thing", "match the visible text", "page_text", "page_html", "SiteNavigationElement", "WPHeader", "WPFooter", "WPSideBar", "cssSelector"} {
		if !strings.Contains(section, want) {
			t.Errorf("Page graph section does not say %q", want)
		}
	}
	if !strings.Contains(guide, "| FAQPage mainEntity |") || strings.Contains(guide, "| mainEntity / FAQPage |") {
		t.Error("guide's FAQ row is not the FAQPage-as-part-of-the-page row")
	}
	for _, want := range []string{"retype none", `{"@id"}`, "Page graph section", "6. build_jsonld(draft, page_html)"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("instructions do not say %q", want)
		}
	}

	cs := connect(t, nil)
	res, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: "entity_schema", Arguments: map[string]string{"keyword": "plumber", "page_url": "http://localhost:8080/p/"}})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Messages[0].Content.(*mcp.TextContent).Text
	if step := selectStep(t, text); !strings.Contains(step, `{"@id"}`) {
		t.Errorf("select_entities step does not say to reference graph nodes by @id: %s", step)
	}
	for _, want := range []string{"5. Shape the graph", "FAQPage that is a part of the page", "6. Call build_jsonld", "as page_html", "## Page graph"} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt does not say %q", want)
		}
	}
}
