// Package mcpserver exposes the schema.org vocabulary and the research
// tools as an MCP server.
package mcpserver

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dazamate/go-json-schema/internal/research"
	"github.com/dazamate/go-json-schema/internal/vocabtool"
)

//go:embed guide.md
var guide string

const instructions = `Builds entity-stuffed schema.org JSON-LD for a keyword, Zizta style.

Workflow:
1. serp(keyword, country, language, near for a city, and any other Google params): top Google results, People also ask, related searches, autocomplete, and bold_terms: the words Google bolds in snippets as matches for the keyword (variants, synonyms, entity names).
2. page_terms(each result's url, or its link where serp gave no url, in rank order, junk such as videos or forums dropped): words and 2-3 word phrases ranked across the pages, the entities competitors already put in their schema, and, when a TextRazor key is configured, the pages' entities and topics linked to Wikipedia in context. rho is each term's rank correlation, as in Cora SEO Software: negative means higher-ranked pages use it more; significant marks the ones that pass the 95% test.
3. select_entities(page_url, keyword): the about and mentions entities for our page, each with a ready thing node. Use them as given: add none, drop none. page_terms entities and resolve_entities lookups no longer feed about or mentions; use resolve_entities for other entity properties such as areaServed.
4. type_properties, enum_values, property_info: every property a type takes, so nothing applicable is left out.
5. build_jsonld(draft): errors to fix and the final script tag.

serp, page_terms, resolve_entities and select_entities fetch through the ChromiumFish browser and fail with install steps if it is missing. The only calls outside it are page_terms and select_entities sending page text they already fetched to TextRazor, and select_entities sending it to Azure AI Language when that is configured. Every request to Google (a search, or a Google redirect link that is followed) waits for a cooldown since the previous one, shared by all schema-mcp processes, so Google is never hit in bursts. Cached results skip it. The vocabulary tools work offline. The stuffing guide is the stuffing_guide prompt.`

// New returns the MCP server. A nil researcher leaves the web tools out.
func New(vocab *vocabtool.Tool, res *research.Researcher, version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "schema", Version: version}, &mcp.ServerOptions{Instructions: instructions})
	addVocabTools(s, vocab)
	if res != nil {
		addResearchTools(s, res)
	}
	addPrompts(s)
	return s
}

type typeArgs struct {
	Type string `json:"type" jsonschema:"schema.org type name, e.g. Recipe or LocalBusiness"`
}

type enumArgs struct {
	Enum string `json:"enum" jsonschema:"schema.org enumeration name, e.g. ItemAvailability or RestrictedDiet"`
}

type propArgs struct {
	Property string `json:"property" jsonschema:"schema.org property name, e.g. knowsAbout"`
}

type buildArgs struct {
	Draft  any  `json:"draft" jsonschema:"JSON-LD draft: one node object, an object with @graph, or an array of nodes. A JSON string is accepted too."`
	Pretty bool `json:"pretty,omitempty" jsonschema:"indent the JSON inside the script tag"`
}

func addVocabTools(s *mcp.Server, v *vocabtool.Tool) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "type_properties",
		Description: "Every property usable on a schema.org type, inherited ones included, with expected types, the class each comes from and the enumeration for enum-only properties. Also the type's ancestors, subtypes and Go type in package schema.",
		Annotations: readOnly(false),
	}, func(_ context.Context, _ *mcp.CallToolRequest, in typeArgs) (*mcp.CallToolResult, vocabtool.TypeInfo, error) {
		info, err := v.TypeProperties(strings.TrimSpace(in.Type))
		return nil, info, err
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "enum_values",
		Description: "Members of a schema.org enumeration, sub-enumerations included, with their URLs, and the properties that expect it.",
		Annotations: readOnly(false),
	}, func(_ context.Context, _ *mcp.CallToolRequest, in enumArgs) (*mcp.CallToolResult, vocabtool.EnumInfo, error) {
		info, err := v.EnumValues(strings.TrimSpace(in.Enum))
		return nil, info, err
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "property_info",
		Description: "A schema.org property's description, domains (types it is listed for), expected types, enumeration members, and its Go field and block in package schema.",
		Annotations: readOnly(false),
	}, func(_ context.Context, _ *mcp.CallToolRequest, in propArgs) (*mcp.CallToolResult, vocabtool.PropInfo, error) {
		info, err := v.PropertyInfo(strings.TrimSpace(in.Property))
		return nil, info, err
	})
	mcp.AddTool(s, &mcp.Tool{
		Name: "build_jsonld",
		Description: "Check a JSON-LD draft against the schema.org vocabulary and return it as a script tag. Unknown types, properties and enumeration members are errors (with suggestions). " +
			"A property used outside its listed domains is only info and is kept: schema.org says domains are closer to guidelines than formal rules. Bare enum names such as InStock become full schema.org URLs. Key order is kept.",
		Annotations: readOnly(false),
	}, func(_ context.Context, req *mcp.CallToolRequest, _ buildArgs) (*mcp.CallToolResult, vocabtool.Report, error) {
		draft, pretty, err := rawDraft(req.Params.Arguments)
		if err != nil {
			return nil, vocabtool.Report{}, err
		}
		rep, err := v.Check(draft, vocabtool.CheckOptions{Pretty: pretty})
		return nil, rep, err
	})
}

// rawDraft reads the draft straight from the arguments, since decoding it
// into Go maps would lose its key order.
func rawDraft(args json.RawMessage) (json.RawMessage, bool, error) {
	var in struct {
		Draft  json.RawMessage `json:"draft"`
		Pretty bool            `json:"pretty"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, false, err
	}
	if len(in.Draft) == 0 || string(in.Draft) == "null" {
		return nil, false, errors.New("draft is empty")
	}
	var text string
	if json.Unmarshal(in.Draft, &text) == nil {
		return json.RawMessage(text), in.Pretty, nil
	}
	return in.Draft, in.Pretty, nil
}

type serpArgs struct {
	Keyword         string            `json:"keyword" jsonschema:"the search phrase"`
	Country         string            `json:"country,omitempty" jsonschema:"country to search from (Google gl), a two-letter code such as us, au or gb; default us"`
	Language        string            `json:"language,omitempty" jsonschema:"interface language (Google hl), e.g. en, de or pt-BR; default en"`
	GoogleDomain    string            `json:"google_domain,omitempty" jsonschema:"Google domain to search on, e.g. google.com.au; default google.com"`
	Page            int               `json:"page,omitempty" jsonschema:"results page, 1 to 10; default 1. Ranks continue from earlier pages"`
	TimeRange       string            `json:"time_range,omitempty" jsonschema:"only results from the past hour, day, week, month or year (Google tbs=qdr)"`
	SafeSearch      bool              `json:"safe_search,omitempty" jsonschema:"filter explicit results (Google safe=active)"`
	ResultsLanguage string            `json:"results_language,omitempty" jsonschema:"only pages written in this language (Google lr), e.g. en"`
	ResultsCountry  string            `json:"results_country,omitempty" jsonschema:"only pages from this country (Google cr), a two-letter code such as au"`
	NoAutocorrect   bool              `json:"no_autocorrect,omitempty" jsonschema:"search the keyword exactly as typed, without Google's spelling correction (nfpr=1)"`
	ShowOmitted     bool              `json:"show_omitted,omitempty" jsonschema:"include results Google leaves out as very similar (filter=0)"`
	WebOnly         bool              `json:"web_only,omitempty" jsonschema:"Google's Web filter (udm=14): plain links without AI Overview or other features; People also ask may be missing"`
	Near            string            `json:"near,omitempty" jsonschema:"place to localize results to, e.g. Austin, TX (Google near=, like Cora's near button). Without it Google localizes to the searching IP's area when country matches it"`
	Params          map[string]string `json:"params,omitempty" jsonschema:"any other Google search URL parameters, set after the named ones, e.g. {\"tbs\": \"cdr:1,cd_min:1/1/2026,cd_max:6/30/2026\"}; q is reserved"`
	ResolveURLs     bool              `json:"resolve_urls,omitempty" jsonschema:"follow each Google redirect link to report exact result URLs. Each one is a Google request and waits for the Google cooldown"`
}

type pageArgs struct {
	Links    []string `json:"links" jsonschema:"each result's url from serp, or its link where serp gave no url, in rank order (at most 20). A Google redirect link waits for the Google cooldown"`
	MaxTerms int      `json:"max_terms,omitempty" jsonschema:"terms per list; default 40"`
	MinPages int      `json:"min_pages,omitempty" jsonschema:"pages a term must appear on; default 2 when 3 or more pages load, else 1"`
}

type entityArgs struct {
	Phrases  []string `json:"phrases" jsonschema:"phrases to look up, at most 100; e.g. keywords and phrases from serp and page_terms"`
	Language string   `json:"language,omitempty" jsonschema:"Wikipedia edition, e.g. en, de; default en"`
}

type selectArgs struct {
	URL                  string   `json:"url" jsonschema:"our page: any URL the browser can load, including localhost"`
	Keyword              string   `json:"keyword" jsonschema:"the page's target keyword"`
	Language             string   `json:"language,omitempty" jsonschema:"Wikipedia edition for titles and sitelinks; default the page's lang, else en"`
	MinRelevance         *float64 `json:"min_relevance,omitempty" jsonschema:"lowest TextRazor relevance for a page entity, 0 to 1; default 0.3"`
	MinTopic             *float64 `json:"min_topic,omitempty" jsonschema:"lowest topic score for a page entity, 0 to 1; default 0.5"`
	MinKeywordSimilarity *float64 `json:"min_keyword_similarity,omitempty" jsonschema:"lowest similarity between a theme and the keyword, 0 to 1; default 0.40"`
	MinPageSimilarity    *float64 `json:"min_page_similarity,omitempty" jsonschema:"lowest similarity between a theme and the page, 0 to 1; default 0.55"`
}

// settings applies the cutoffs that were passed over the defaults.
func (a selectArgs) settings() research.SelectSettings {
	st := research.DefaultSelectSettings
	for _, c := range []struct {
		in  *float64
		out *float64
	}{{a.MinRelevance, &st.MinRelevance}, {a.MinTopic, &st.MinTopic}, {a.MinKeywordSimilarity, &st.MinKeywordSimilarity}, {a.MinPageSimilarity, &st.MinPageSimilarity}} {
		if c.in != nil {
			*c.out = *c.in
		}
	}
	return st
}

func addResearchTools(s *mcp.Server, r *research.Researcher) {
	wait := cooldownNote(r.Google.Interval())
	mcp.AddTool(s, &mcp.Tool{
		Name: "serp",
		Description: "Search Google for a keyword through the ChromiumFish stealth browser: top 10 organic results (title, domain, snippet, link), People also ask questions, related searches and autocomplete suggestions. " +
			"bold_terms are the short words and phrases Google bolds in the snippets as matches for the keyword (variants, synonyms, entity names), with how many results bold each; bold_passages are the longer answers it bolds. " +
			"Takes the country and language to search as (default us, en), near to localize to a city, and Google's other search params: domain, page, time range, safe search, results language and country, no autocorrect, omitted results, the Web filter, and any raw URL params. Personalization is always off (pws=0). " +
			"Falls back to DuckDuckGo, then Bing, when Google shows a bot wall; engine says which answered and note lists the Google-only params it ignored. search_url is the search made. Cached for a day." + wait,
		Annotations: readOnly(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in serpArgs) (*mcp.CallToolResult, research.SERP, error) {
		out, err := r.SERP(withProgress(ctx, req), research.SERPQuery{
			Keyword: in.Keyword, Country: in.Country, Language: in.Language, Domain: in.GoogleDomain, Page: in.Page,
			TimeRange: in.TimeRange, SafeSearch: in.SafeSearch, ResultsLanguage: in.ResultsLanguage, ResultsCountry: in.ResultsCountry,
			NoAutocorrect: in.NoAutocorrect, ShowOmitted: in.ShowOmitted, WebOnly: in.WebOnly, Near: in.Near, Params: in.Params, ResolveURLs: in.ResolveURLs,
		})
		return nil, out, err
	})
	mcp.AddTool(s, &mcp.Tool{
		Name: "page_terms",
		Description: "Read ranking pages through the browser and rank the single words and 2-3 word phrases they share, like Zizta's semantic keywords: count, pages using it, and a relevance score favouring widely used terms and title or heading hits. " +
			"With 5 or more pages each term has rho, its Spearman correlation between use and rank as in Cora SEO Software: negative means higher-ranked pages use it more, and significant marks |rho| passing the 95% test. " +
			"With a TextRazor key configured, entities are the pages' mentions linked to Wikipedia in context (name is the exact Wikipedia title, so pass it to resolve_entities; also qid, the words used, mentions, pages, relevance, rho) and topics are the pages' themes; entity_note gives TextRazor requests used (free plan: 500 a day) or why there are none. " +
			"Also each page's title, description, headings and a summary of its JSON-LD (types, about, mentions, knowsAbout, keywords), merged across pages in schema_terms. Pages and TextRazor results are cached for a week. " +
			"Pass each result's url where serp gave one: a Google redirect link is a Google request that waits for the Google cooldown, so those are read last." + wait,
		Annotations: readOnly(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in pageArgs) (*mcp.CallToolResult, research.PageReport, error) {
		out, err := r.PageTerms(withProgress(ctx, req), research.PageQuery{Links: in.Links, MaxTerms: in.MaxTerms, MinPages: in.MinPages})
		return nil, out, err
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:         "resolve_entities",
		OutputSchema: outputSchema[research.EntityReport](),
		Description: "Look phrases up as Wikipedia articles (following redirects) and Wikidata items: QID, label, description, aliases, Freebase /m/ ID (P646), Google KG /g/ ID (P2671), types, and a ready schema.org Thing node with every sameAs URL and its GoogleMachineID. " +
			"status is exact, redirect, section (the phrase is a section of a broader article), ambiguous or missing; the last two list candidates to retry with. Never invent IDs a lookup did not return. Cached for 30 days.",
		Annotations: readOnly(true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in entityArgs) (*mcp.CallToolResult, research.EntityReport, error) {
		out, err := r.ResolveEntities(ctx, research.EntityQuery{Phrases: in.Phrases, Language: in.Language})
		return nil, out, err
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:         "select_entities",
		OutputSchema: outputSchema[research.SelectReport](),
		Description: "Choose the about and mentions entities for OUR page and its target keyword. The page is read through the browser (localhost works), its title and text are linked to Wikipedia and Wikidata by TextRazor in context, and every entity is checked by its Wikidata item. " +
			"about is the keyword's own entities (found on the page, and looked up as exact Wikipedia titles from the keyword's words and phrases) plus the page's themes, which the page may not name. mentions holds only entities the page names. There are no caps on either. " +
			"Each entry comes with its ready schema.org thing node (Wikipedia, Wikidata and Google kgmid sameAs, Bing ID, alternateName from the words the page uses, additionalType): use it as given. dropped lists what was rejected and why, so a looser cutoff can be judged. " +
			"The four defaults (min_relevance 0.3, min_topic 0.5, min_keyword_similarity 0.40, min_page_similarity 0.55) are our judgement, set by eye from one calibration page, not from any Google source; each is a number from 0 to 1. " +
			"It needs a TextRazor key (textrazor.api_key; free plan: 500 requests a day) and fails without one. An Azure AI Language key (Bing IDs) and the local embedding model (themes; about 220 MB downloaded on first use) are optional, and note says which are missing and why. " +
			"The page text goes to TextRazor and, when configured, Azure; Wikipedia and Wikidata calls go through the browser. Results are cached for a week by the text sent, but the page itself is always read fresh.",
		Annotations: readOnly(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in selectArgs) (*mcp.CallToolResult, research.SelectReport, error) {
		out, err := r.SelectEntities(withProgress(ctx, req), research.SelectQuery{URL: in.URL, Keyword: in.Keyword, Language: in.Language, Settings: in.settings()})
		return nil, out, err
	})
}

// cooldownNote tells the model how long Google requests wait.
func cooldownNote(interval time.Duration) string {
	if interval <= 0 {
		return ""
	}
	return fmt.Sprintf(" Google requests are spaced at least %s apart across every schema-mcp process, so each one may wait up to %[1]s first.", interval)
}

// withProgress sends what the research call is doing, such as waiting for
// the Google cooldown, as progress notifications when the client asked for
// them. They also keep a client's idle timeout from ending a long call.
func withProgress(ctx context.Context, req *mcp.CallToolRequest) context.Context {
	token := req.Params.GetProgressToken()
	if token == nil || req.Session == nil {
		return ctx
	}
	var n atomic.Int64
	return research.WithProgress(ctx, func(msg string) {
		req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{ProgressToken: token, Progress: float64(n.Add(1)), Message: msg})
	})
}

// outputSchema infers T's schema with pre-encoded JSON (an entity's
// thing node) as an object rather than a byte array.
func outputSchema[T any]() *jsonschema.Schema {
	sch, err := jsonschema.For[T](&jsonschema.ForOptions{TypeSchemas: map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[json.RawMessage](): {Type: "object"},
	}})
	if err != nil {
		panic(err)
	}
	return sch
}

func readOnly(openWorld bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld}
}

func addPrompts(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "stuffing_guide",
		Description: "How hard to stuff each schema.org property, and where its URLs should come from.",
	}, func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: guide}}}}, nil
	})
	s.AddPrompt(&mcp.Prompt{
		Name:        "entity_schema",
		Description: "Research a keyword and build entity-stuffed JSON-LD for a page targeting it.",
		Arguments: []*mcp.PromptArgument{
			{Name: "keyword", Description: "the page's target keyword", Required: true},
			{Name: "page_type", Description: "main schema.org type, e.g. Recipe, LocalBusiness, Article; default WebPage"},
			{Name: "country", Description: "country to search from, a two-letter code such as us or au; default us"},
			{Name: "language", Description: "language to search in, e.g. en or pt-BR; default en"},
		},
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		a := req.Params.Arguments
		if strings.TrimSpace(a["keyword"]) == "" {
			return nil, errors.New("keyword is required")
		}
		if strings.TrimSpace(a["page_url"]) == "" {
			return nil, errors.New("page_url is required: select_entities reads our page for about and mentions")
		}
		pageType := orDefault(a["page_type"], "WebPage")
		country, lang := orDefault(a["country"], "us"), orDefault(a["language"], "en")
		wiki, _, _ := strings.Cut(strings.ToLower(lang), "-")
		text := fmt.Sprintf(`Build over-the-top schema.org JSON-LD for a page targeting %q (main type %s, country %s, language %s).

1. Call serp with keyword %[1]q, country %[3]s and language %[4]s (add near for a city when the page is local). Note People also ask questions for an FAQ, related searches as keywords, and bold_terms as the variants Google matches for the keyword.
2. Call page_terms with each organic result's url, or its link where serp gave no url, in rank order, leaving out videos, forums and other junk. Favour terms whose rho is negative and significant: higher-ranked pages use them more. Its entities and topics show what competitors cover: use them for keywords and the page's copy, not for about or mentions.
3. Call select_entities with url %[6]q, keyword %[1]q and language %[5]s. Put its about and mentions in the graph as given.
4. Call type_properties for %[2]s and fill every property that fits: keywords, knowsAbout, alternateName, additionalType, disambiguatingDescription and the type's own properties. Use resolve_entities for other entity properties such as areaServed.
5. Call build_jsonld with the draft, fix every error, and return the script tag.

%[7]s`, a["keyword"], pageType, country, lang, wiki, strings.TrimSpace(a["page_url"]), guide)
		return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}}}, nil
	})
}

func orDefault(s, def string) string {
	if s = strings.TrimSpace(s); s == "" {
		return def
	}
	return s
}
