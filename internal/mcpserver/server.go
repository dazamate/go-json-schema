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
3. select_entities(page_url, keyword): the about and mentions entities for our page, each with a ready thing node typed from its Wikidata class (Language, City, Country, Occupation, Organization, Person and the like; a general idea stays a Thing). Use them as given: add none, drop none, retype none. The one exception: when an entity is the same Wikidata item as a node already in the graph (a served area, a language, the person's occupation), put that node's {"@id"} in its place. page_terms entities and resolve_entities lookups no longer feed about or mentions; use resolve_entities for other entity properties such as areaServed.
4. type_properties, enum_values, property_info: every property a type takes, so nothing applicable is left out.
5. Shape the graph as the stuffing guide's Page graph section says: the sitewide nodes on every page, the page's main entity, an FAQ as an FAQPage that is a part of the page, @id references for repeated nodes, and only the actions the page supports.
6. build_jsonld(draft, page_html): errors to fix, warnings to weigh, the page's sections to mark up, and the final script tag. Give it the rendered page as page_html (or only its visible text as page_text). It warns on any about or mentions entity the page does not name, and on any nav, page header or footer, aside, table or ad slot the draft does not mark up. It holds the markup to what the page shows, both ways: the step, question, answer and review text it repeats must be on the page, and the page's prices, ratings, quotes, terms, images, videos, copyright line, reading time, counted lists, outside and related links, forms, contact links and registration numbers must be in it. It also warns on a repeated Wikidata item, a node a more specific subtype fits (a property only the subtype takes, a Wikidata class that gives it, or a name or URL path that names it), and a plain WebPageElement a more specific type fits.

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
	Draft    any    `json:"draft" jsonschema:"JSON-LD draft: one node object, an object with @graph, or an array of nodes. A JSON string is accepted too."`
	Pretty   bool   `json:"pretty,omitempty" jsonschema:"indent the JSON inside the script tag"`
	PageText string `json:"page_text,omitempty" jsonschema:"the page's visible text; when given, every about and mentions entity must be named in it"`
	PageHTML string `json:"page_html,omitempty" jsonschema:"the rendered page's HTML; its navs, page header and footer, asides, tables and ad slots must be marked up, and it gives page_text when that is not given"`
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
			"A property used outside its listed domains is only info and is kept: schema.org says domains are closer to guidelines than formal rules. These are warnings: a property that a subtype of the node's type takes, though the type does not (type the node as that subtype: openingHours makes an Organization a LocalBusiness); a node whose first Wikidata class in additionalType gives a subtype of its type, when none of its other classes gives its type (type it so, unless the item is a kind of thing); a node whose name, headline or URL path names a subtype of its type (a WebPage at /about/ is an AboutPage); a node without an @id that names the same Wikidata item as a node with one (reference that @id instead); a plain WebPageElement that a more specific type fits (a section of Questions is an FAQPage that is a part of the page); and, with page_text or page_html, an about or mentions entity the page does not name. With page_html it also holds the markup to the page: HowToStep, Question, Answer and Review text must be on the page and every #fragment an id on it, and the page's prices (Offer or AggregateOffer, with valueAddedTaxIncluded), ratings, blockquotes (Review or Quotation), terms in another script (DefinedTerm), content images (ImageObject with width and height), video and audio embeds, copyright line, reading time (timeRequired), section label (articleSection), counted headings (ItemList), groups of outside links, related-reading links (relatedLink), reply-time promises (processingTime), search box (SearchAction), forms (an Action targeting them, with httpMethod), tel: and mailto: links and registration numbers (taxID) must be marked up. A Review of the page's own business is only info: valid, though Google shows no stars for it. With page_html the report lists the page's sections that a specific type describes (navs as SiteNavigationElement, the page header and footer as WPHeader and WPFooter, asides as WPSideBar, tables as Table, ad slots as WPAdBlock, a breadcrumb nav as BreadcrumbList) with a CSS selector and the node that marks each up, and warns on every one left unmarked and on a cssSelector that selects nothing or a section of another type. Bare enum names such as InStock become full schema.org URLs. Key order is kept.",
		Annotations: readOnly(false),
	}, func(_ context.Context, req *mcp.CallToolRequest, _ buildArgs) (*mcp.CallToolResult, vocabtool.Report, error) {
		draft, opts, err := rawDraft(req.Params.Arguments)
		if err != nil {
			return nil, vocabtool.Report{}, err
		}
		rep, err := v.Check(draft, opts)
		return nil, rep, err
	})
}

// rawDraft reads the draft straight from the arguments, since decoding it
// into Go maps would lose its key order, and the check options beside it.
func rawDraft(args json.RawMessage) (json.RawMessage, vocabtool.CheckOptions, error) {
	var in struct {
		Draft    json.RawMessage `json:"draft"`
		Pretty   bool            `json:"pretty"`
		PageText string          `json:"page_text"`
		PageHTML string          `json:"page_html"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, vocabtool.CheckOptions{}, err
	}
	opts := vocabtool.CheckOptions{Pretty: in.Pretty, PageText: in.PageText, PageHTML: in.PageHTML}
	if len(in.Draft) == 0 || string(in.Draft) == "null" {
		return nil, opts, errors.New("draft is empty")
	}
	var text string
	if json.Unmarshal(in.Draft, &text) == nil {
		return json.RawMessage(text), opts, nil
	}
	return in.Draft, opts, nil
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
	MinLinkConfidence    *float64 `json:"min_link_confidence,omitempty" jsonschema:"lowest TextRazor confidence (about 0.5 to 10) for an entity kept only by a link to another page of our site, 0 or more; default 1.5"`
	MinGraphSimilarity   *float64 `json:"min_graph_similarity,omitempty" jsonschema:"lowest similarity between a linked Wikipedia article and the keyword, 0 to 1, for an article the page does not name; default 0.40"`
	MinGraphLinks        *int     `json:"min_graph_links,omitempty" jsonschema:"links into the graph a page entity that qualifies on its own (relevance, topic score or own-site link) needs to join about, 1 or more; default 3"`
	MinGraphLinksUngated *int     `json:"min_graph_links_ungated,omitempty" jsonschema:"links into the graph a page entity that does not qualify on its own, or a keyword lookup hit TextRazor did not link, needs to join about, 1 or more; default 10"`
	MinMentionGraphLinks *int     `json:"min_mention_graph_links,omitempty" jsonschema:"links into the graph a page entity that qualifies on its own needs to stay in mentions (it must also be named in the page text), 0 or more; default 2"`
	MaxGraphNeighbours   *int     `json:"max_graph_neighbours,omitempty" jsonschema:"most linked articles the page does not name that join about, 0 or more; default 30"`
}

// settings applies the cutoffs that were passed over the defaults.
func (a selectArgs) settings() research.SelectSettings {
	st := research.DefaultSelectSettings
	for _, c := range []struct {
		in  *float64
		out *float64
	}{{a.MinRelevance, &st.MinRelevance}, {a.MinTopic, &st.MinTopic}, {a.MinKeywordSimilarity, &st.MinKeywordSimilarity}, {a.MinPageSimilarity, &st.MinPageSimilarity}, {a.MinLinkConfidence, &st.MinLinkConfidence}, {a.MinGraphSimilarity, &st.MinGraphSimilarity}} {
		if c.in != nil {
			*c.out = *c.in
		}
	}
	for _, c := range []struct {
		in  *int
		out *int
	}{{a.MinGraphLinks, &st.MinGraphLinks}, {a.MinGraphLinksUngated, &st.MinGraphLinksUngated}, {a.MinMentionGraphLinks, &st.MinMentionGraphLinks}, {a.MaxGraphNeighbours, &st.MaxGraphNeighbours}} {
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
		Description: "Look phrases up as Wikipedia articles (following redirects) and Wikidata items: QID, label, description, aliases, Freebase /m/ ID (P646), Google KG /g/ ID (P2671), types, and a ready schema.org node with every sameAs URL and its GoogleMachineID, typed from the item's Wikidata class (P31) when it is one place, language, occupation, organization or person, and a Thing otherwise. " +
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
			"When the keyword has its own Wikipedia article (the whole keyword, or a phrase of 2 or more of its words), about is instead that article, the articles linked to or from it that are close to the keyword or named on the page (at most max_graph_neighbours unnamed ones), and the page entities well linked into that graph; mentions then holds only page entities linked into it, and graph reports its size. " +
			"Each entry comes with its ready schema.org thing node (Wikipedia, Wikidata and Google kgmid sameAs, Bing ID, alternateName from the words the page uses, additionalType), typed from the item's Wikidata class (schema_type: Language, City, Country, Occupation, Organization, Person and the like; a general idea stays a Thing): use it as given, except that an entity already in the graph as a node with an @id is referenced by that @id. dropped lists what was rejected and why, so a looser cutoff can be judged. " +
			"The defaults (min_relevance 0.3, min_topic 0.5, min_keyword_similarity 0.40, min_page_similarity 0.55, min_link_confidence 1.5, min_graph_similarity 0.40, min_graph_links 3, min_graph_links_ungated 10, min_mention_graph_links 2, max_graph_neighbours 30) are our judgement, set by eye from two calibration pages (the moving page for the base settings, the forex page for the five graph settings), not from any Google source (the cap was set by the owner); each similarity is a number from 0 to 1, except min_link_confidence, a TextRazor confidence of 0 or more that an entity kept only by a link to another page of our site must reach. min_graph_links and min_graph_links_ungated are whole numbers of 1 or more, and min_mention_graph_links and max_graph_neighbours are whole numbers of 0 or more. A page entity that qualifies on its own needs min_graph_links links to join about; one that does not, and a keyword lookup hit TextRazor did not link, needs min_graph_links_ungated. " +
			"It needs a TextRazor key (textrazor.api_key; free plan: 500 requests a day) and fails without one. An Azure AI Language key (Bing IDs) and the local embedding model (it scores the page's themes and the graph's neighbours; without it the graph keeps only the neighbours the page names; about 220 MB downloaded on first use) are optional, and note says which are missing and why. " +
			"The page text goes to TextRazor and, when configured, Azure; Wikipedia and Wikidata calls go through the browser. Results are cached for a week by the text sent, and the graph neighbourhood of a keyword article for 30 days, but the page itself is always read fresh.",
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
			{Name: "page_url", Description: "our page's URL (localhost works); select_entities reads it for about and mentions", Required: true},
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
		pageURL := strings.TrimSpace(a["page_url"])
		// select_entities defaults to the page's lang, so it gets a language
		// only when one was given.
		selectWith := fmt.Sprintf("url %q and keyword %q", pageURL, a["keyword"])
		if asked := strings.TrimSpace(a["language"]); asked != "" {
			wiki, _, _ := strings.Cut(strings.ToLower(asked), "-")
			selectWith = fmt.Sprintf("url %q, keyword %q and language %s", pageURL, a["keyword"], wiki)
		}
		text := fmt.Sprintf(`Build over-the-top schema.org JSON-LD for a page targeting %q (main type %s, country %s, language %s).

1. Call serp with keyword %[1]q, country %[3]s and language %[4]s (add near for a city when the page is local). Note People also ask questions for an FAQ, related searches as keywords, and bold_terms as the variants Google matches for the keyword.
2. Call page_terms with each organic result's url, or its link where serp gave no url, in rank order, leaving out videos, forums and other junk. Favour terms whose rho is negative and significant: higher-ranked pages use them more. Its entities and topics show what competitors cover: use them for keywords and the page's copy, not for about or mentions.
3. Call select_entities with %[5]s. Put its about and mentions in the graph as given, typed as they come; where an entity is the same Wikidata item as a node already in the graph, put that node's {"@id"} in its place.
4. Call type_properties for %[2]s and fill every property that fits: keywords, knowsAbout, alternateName, additionalType, disambiguatingDescription and the type's own properties. Use resolve_entities for other entity properties such as areaServed.
5. Shape the graph as the Page graph section below says: the WebSite, business, Person, Brand and BreadcrumbList on every page, the WebPage's mainEntity and its mainEntityOfPage back, the FAQ as an FAQPage that is a part of the page (hasPart and isPartOf), @id references for repeated nodes, and potentialAction only for what the page lets a visitor do.
6. Call build_jsonld with the draft and the rendered page as page_html. Fix every error, weigh every warning (retype a node to the subtype a warning names when it is true of the node), mark up every price, quote, image, term and list the page shows that a warning names, mark up every section its page_elements leave unmarked, and return the script tag.

%[6]s`, a["keyword"], pageType, country, lang, selectWith, guide)
		return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}}}, nil
	})
}

func orDefault(s, def string) string {
	if s = strings.TrimSpace(s); s == "" {
		return def
	}
	return s
}
