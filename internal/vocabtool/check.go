package vocabtool

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

// Issue severities. Only errors stop a draft being valid schema.org.
const (
	Error   = "error"
	Warning = "warning"
	Info    = "info"
)

// Issue is one finding about a draft, located by a JSONPath-like path, or for
// a finding on the page itself, by "html:" and the element's CSS selector.
type Issue struct {
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Report is the checked draft, ready to publish, plus what was found.
type Report struct {
	Errors       int           `json:"errors"`
	Warnings     int           `json:"warnings"`
	Issues       []Issue       `json:"issues"`
	PageElements []PageElement `json:"page_elements,omitempty"`
	ScriptTag    string        `json:"script_tag"`
}

// PageElement is a section of the page that a specific schema.org type
// describes: a WebPageElement subtype, or a BreadcrumbList for a breadcrumb
// nav. MarkedBy is the path of the draft node that marks it up, "" for none.
type PageElement struct {
	Type     string `json:"type"`
	Selector string `json:"selector"`
	Name     string `json:"name,omitempty"`
	MarkedBy string `json:"marked_by,omitempty"`
}

// CheckOptions controls the published form and the page checks.
type CheckOptions struct {
	Pretty   bool   // indent the JSON inside the script tag
	PageText string // the page's visible text; when set, about and mentions entities must be named in it
	PageHTML string // the rendered page; its sections must be marked up, and it gives PageText when that is empty
	PageURL  string // the page's address; links resolve against it when the page has no canonical link
	// Keyword is the page's target keyword; when set, the page's names,
	// headlines and descriptions must carry it at KeywordDensity.
	Keyword string
	// KeywordDensity is the keyword's target share of the words, in percent;
	// 0 is DefaultKeywordDensity.
	KeywordDensity float64
}

// Check validates a JSON-LD draft against the vocabulary and returns it as
// a script tag. Unknown types, properties and enumeration members are
// errors. A property used on a type outside its listed domains is only
// info: schema.org calls domains "closer to guidelines than formal rules"
// (https://schema.org/docs/datamodel.html), so it is kept. Bare enum member
// names such as InStock are expanded to their schema.org URL.
func (t *Tool) Check(draft json.RawMessage, opts CheckOptions) (Report, error) {
	root, err := jsonld.Decode(draft)
	if err != nil {
		return Report{}, fmt.Errorf("draft is not valid JSON: %w", err)
	}
	var doc *jsonld.Object
	switch r := root.(type) {
	case *jsonld.Object:
		doc = r
		if _, ok := doc.Get("@context"); !ok {
			doc.Prepend("@context", "https://schema.org")
		}
	case []any:
		doc = &jsonld.Object{}
		doc.Set("@context", "https://schema.org")
		doc.Set("@graph", r)
	default:
		return Report{}, fmt.Errorf("draft must be a JSON object or array")
	}
	c := checker{v: t.Vocab, suggested: map[*jsonld.Object]bool{}}
	c.walk("$", doc)
	c.repeats(doc)
	all := jsonld.Nodes("$", doc)
	c.classTypes(all)
	c.namedTypes(all)
	c.webPageElements(all)
	text := opts.PageText
	var elements []PageElement
	var page *html.Node
	if strings.TrimSpace(opts.PageHTML) != "" {
		if page, err = html.Parse(strings.NewReader(opts.PageHTML)); err != nil {
			return Report{}, fmt.Errorf("page_html: %w", err)
		}
		if text == "" {
			text = htmlText(page)
		}
	}
	var facts *pageFacts
	if strings.TrimSpace(text) != "" {
		facts = newPageFacts(page, text, opts.PageURL)
	}
	if page != nil {
		// a page search engines skip still lists its sections, unasked
		elements = c.pageElements(all, page, facts == nil || facts.private == "")
	}
	c.pageEntities(all, text)
	c.bare(all)
	c.ownReviews(all)
	c.reviews(all)
	c.productSnippets(all)
	if facts != nil {
		c.content(all, facts)
	}
	known := facts
	if known == nil {
		known = &pageFacts{}
		if u, err := url.Parse(strings.TrimSpace(opts.PageURL)); err == nil && u.IsAbs() {
			known.url = u
		}
	}
	c.organizations(all, opts.Keyword)
	c.looseNodes(all, known)
	if strings.TrimSpace(opts.Keyword) != "" {
		c.keyword(all, doc, opts.Keyword, opts.KeywordDensity, known)
	}
	rep := Report{Issues: c.issues, PageElements: elements}
	if rep.Issues == nil {
		rep.Issues = []Issue{}
	}
	for _, i := range c.issues {
		switch i.Severity {
		case Error:
			rep.Errors++
		case Warning:
			rep.Warnings++
		}
	}
	body, err := jsonld.Encode(doc, opts.Pretty)
	if err != nil {
		return Report{}, err
	}
	safe := strings.NewReplacer("<", `<`, ">", `>`).Replace(string(body))
	rep.ScriptTag = `<script type="application/ld+json">` + safe + `</script>`
	return rep, nil
}

// Vocabulary checks a draft against the vocabulary alone: the walk Check
// starts with, without its page, site and naming checks. Unknown types,
// properties and enumeration members are errors; superseded terms, an
// untyped node with properties and a property only a subtype takes are
// warnings; a property outside its listed domains is info.
func (t *Tool) Vocabulary(draft json.RawMessage) ([]Issue, error) {
	root, err := jsonld.Decode(draft)
	if err != nil {
		return nil, fmt.Errorf("draft is not valid JSON: %w", err)
	}
	switch root.(type) {
	case *jsonld.Object, []any:
	default:
		return nil, fmt.Errorf("draft must be a JSON object or array")
	}
	c := checker{v: t.Vocab, suggested: map[*jsonld.Object]bool{}}
	c.walk("$", root)
	return c.issues, nil
}

type checker struct {
	v         *vocab.Vocab
	issues    []Issue
	suggested map[*jsonld.Object]bool // nodes already told to take a more specific type
}

func (c *checker) add(path, severity, format string, args ...any) {
	c.issues = append(c.issues, Issue{Path: path, Severity: severity, Message: fmt.Sprintf(format, args...)})
}

// walk checks a value. Objects with @type are nodes; their properties are
// checked against their types.
func (c *checker) walk(path string, v any) {
	switch x := v.(type) {
	case []any:
		for i, item := range x {
			c.walk(fmt.Sprintf("%s[%d]", path, i), item)
		}
	case *jsonld.Object:
		types := c.types(path, x)
		hasProps := false
		for _, k := range x.Keys {
			if strings.HasPrefix(k, "@") {
				if k == "@graph" {
					c.walk(path+".@graph", x.Vals[k])
				}
				continue
			}
			hasProps = true
			c.property(path+"."+k, k, types, x)
		}
		if _, typed := x.Get("@type"); !typed && hasProps {
			c.add(path, Warning, "node has properties but no @type")
		}
	}
}

// googleAnnotation is a property such as query-input that Google's action
// markup adds beside a schema.org Action.
var googleAnnotation = regexp.MustCompile(`^[a-z]+-(?:input|output)$`)

// repeats warns about a node without an @id that names the same Wikidata
// item as a node with one: it repeats that node, which a {"@id"} reference
// says once.
func (c *checker) repeats(doc *jsonld.Object) {
	byItem := map[string]string{}
	var collect func(v any)
	collect = func(v any) {
		switch x := v.(type) {
		case []any:
			for _, item := range x {
				collect(item)
			}
		case *jsonld.Object:
			if id, ok := x.Get("@id"); ok && len(x.Keys) > 1 {
				if s, ok := id.(string); ok {
					for _, q := range jsonld.WikidataItems(x) {
						if _, seen := byItem[q]; !seen {
							byItem[q] = s
						}
					}
				}
			}
			for _, k := range x.Keys {
				collect(x.Vals[k])
			}
		}
	}
	collect(doc)
	if len(byItem) == 0 {
		return
	}
	var check func(path string, v any)
	check = func(path string, v any) {
		switch x := v.(type) {
		case []any:
			for i, item := range x {
				check(fmt.Sprintf("%s[%d]", path, i), item)
			}
		case *jsonld.Object:
			if _, ok := x.Get("@id"); !ok {
				for _, q := range jsonld.WikidataItems(x) {
					if id, ok := byItem[q]; ok {
						c.add(path, Warning, "same Wikidata item (%s) as the node %s: put {\"@id\": %q} here instead of repeating it", q, id, id)
						break
					}
				}
			}
			for _, k := range x.Keys {
				if k != "@context" {
					check(path+"."+k, x.Vals[k])
				}
			}
		}
	}
	check("$", doc)
}

// types returns the node's known schema.org types, reporting unknown ones.
func (c *checker) types(path string, n *jsonld.Object) []string {
	raw, ok := n.Get("@type")
	if !ok {
		return nil
	}
	var names []string
	switch x := raw.(type) {
	case string:
		names = []string{x}
	case []any:
		for _, e := range x {
			if s, ok := e.(string); ok {
				names = append(names, s)
			}
		}
	}
	var known []string
	for _, name := range names {
		term := jsonld.StripPrefix(name)
		cl, ok := c.v.V.Classes[term]
		if !ok {
			c.unknown(path, vocab.KindType, name)
			continue
		}
		if len(cl.SupersededBy) > 0 {
			c.add(path, Warning, "type %s is superseded by %s", term, strings.Join(cl.SupersededBy, ", "))
		}
		known = append(known, term)
	}
	return known
}

func (c *checker) unknown(path, kind, name string) {
	if s := c.v.Suggest(kind, jsonld.StripPrefix(name)); s != "" {
		c.add(path, Error, "unknown %s %q; did you mean %s?", kind, name, s)
		return
	}
	c.add(path, Error, "unknown %s %q", kind, name)
}

func (c *checker) property(path, key string, types []string, n *jsonld.Object) {
	term := jsonld.StripPrefix(key)
	if googleAnnotation.MatchString(term) {
		c.add(path, Info, "%s is Google's action annotation, not a schema.org property; kept", key)
		return
	}
	if strings.Contains(term, ":") || strings.Contains(term, "/") {
		c.add(path, Warning, "%q is not a schema.org term; left as is", key)
		return
	}
	p, ok := c.v.V.Props[term]
	if !ok {
		c.unknown(path, vocab.KindProperty, key)
		return
	}
	if len(p.SupersededBy) > 0 {
		c.add(path, Warning, "property %s is superseded by %s", term, strings.Join(p.SupersededBy, ", "))
	}
	if numericOnly(p.Ranges) {
		for _, v := range jsonld.Values(n.Vals[key]) {
			if s, ok := v.(string); ok {
				if _, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
					c.add(path, Warning, "%s takes a number: write %s, not %q", term, strings.TrimSpace(s), s)
				}
			}
		}
	}
	if len(types) > 0 && !slices.ContainsFunc(types, func(ty string) bool { return c.v.Applicable(ty, term) }) {
		// a subtype that takes the property is the type the node means
		if subs := c.v.SubtypesTaking(types, term); len(subs) > 0 {
			noun := "subtype"
			if len(subs) > 1 {
				noun = "subtypes"
			}
			c.add(path, Warning, "%s is not listed for %s but is for its %s %s: type it %s", term, strings.Join(types, ", "), noun, strings.Join(subs, " and "), strings.Join(subs, " or "))
			c.suggested[n] = true
		} else {
			c.add(path, Info, "%s is not listed for %s (domains: %s); kept, since schema.org domains are guidelines rather than rules", term, strings.Join(types, ", "), strings.Join(p.Domains, ", "))
		}
	}
	val := n.Vals[key]
	switch x := val.(type) {
	case []any:
		for i, item := range x {
			x[i] = c.value(fmt.Sprintf("%s[%d]", path, i), term, item)
		}
	default:
		n.Vals[key] = c.value(path, term, val)
	}
}

// value checks one property value: enumeration members by name, nested
// nodes recursively.
func (c *checker) value(path, prop string, v any) any {
	s, ok := v.(string)
	if !ok {
		c.walk(path, v)
		return v
	}
	if enum, only := c.v.EnumOnly(prop); only {
		name := jsonld.StripPrefix(s)
		if slices.Contains(c.v.Members(enum), name) {
			return baseURL + name
		}
		allowed := c.v.Members(enum)
		if len(allowed) > 15 {
			allowed = append(allowed[:15:15], "...")
		}
		c.add(path, Error, "%q is not an %s member; allowed: %s", s, enum, strings.Join(allowed, ", "))
		return v
	}
	// Mixed ranges such as Text|ItemListOrderType: only a schema.org URL
	// claims to be a member, so only that is checked.
	if name, isURL := jsonld.CutSchemaURL(s); isURL {
		if _, isMember := c.v.V.Members[name]; !isMember && c.v.RangesEnum(prop) {
			c.add(path, Warning, "%q names no schema.org enumeration member", s)
		}
	}
	return v
}
