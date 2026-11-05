package audit

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
)

// Input is what a caller gives the audit beside the draft.
type Input struct {
	PageText string // the page's visible text; "" takes it from PageHTML
	PageHTML string // the rendered page
	PageURL  string // the page's address
	Keyword  string // the page's target keyword
	// KeywordDensity is the keyword's target share of the words, in percent;
	// 0 is the default.
	KeywordDensity float64
}

// Context is what the checks read: the vocabulary, the normalized draft, the
// page and the caller's input. It also memoizes the facts several checks
// share. A Context is not safe for concurrent use; the checks of one audit
// run one after another.
type Context struct {
	Vocab *vocab.Vocab
	Root  any            // the normalized draft root: a *jsonld.Object, or a []any for Vocabulary
	Nodes []jsonld.Node  // every object in the draft, outermost first
	HTML  *html.Node     // the parsed page_html, nil without it
	Page  *webpage.Facts // the page, nil without page text or html
	Known *webpage.Facts // Page, or else facts holding only the page's URL
	In    Input

	memo map[any]any
}

// NewContext decodes a draft for Check. An object root gets a schema.org
// @context when it has none; an array root becomes the @graph of one. The
// draft is normalized, and the page is parsed and read when it is given.
func NewContext(v *vocab.Vocab, draft []byte, in Input) (*Context, error) {
	root, err := jsonld.Decode(draft)
	if err != nil {
		return nil, fmt.Errorf("draft is not valid JSON: %w", err)
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
		return nil, fmt.Errorf("draft must be a JSON object or array")
	}
	Normalize(v, doc)
	c := &Context{Vocab: v, Root: doc, Nodes: jsonld.Nodes("$", doc), In: in}
	text := in.PageText
	if strings.TrimSpace(in.PageHTML) != "" {
		if c.HTML, err = html.Parse(strings.NewReader(in.PageHTML)); err != nil {
			return nil, fmt.Errorf("page_html: %w", err)
		}
		if text == "" {
			text = webpage.HTMLText(c.HTML)
		}
	}
	if strings.TrimSpace(text) != "" {
		c.Page = webpage.New(c.HTML, text, in.PageURL)
	}
	c.Known = c.Page
	if c.Known == nil {
		c.Known = &webpage.Facts{}
		if u, err := url.Parse(strings.TrimSpace(in.PageURL)); err == nil && u.IsAbs() {
			c.Known.URL = u
		}
	}
	return c, nil
}

// NewVocabularyContext decodes a draft for Vocabulary: normalized, but with
// no @context added and an array root kept as given, so its nodes keep their
// $[i] paths. It has no page.
func NewVocabularyContext(v *vocab.Vocab, draft []byte) (*Context, error) {
	root, err := jsonld.Decode(draft)
	if err != nil {
		return nil, fmt.Errorf("draft is not valid JSON: %w", err)
	}
	switch root.(type) {
	case *jsonld.Object, []any:
	default:
		return nil, fmt.Errorf("draft must be a JSON object or array")
	}
	Normalize(v, root)
	return &Context{Vocab: v, Root: root, Nodes: jsonld.Nodes("$", root), Known: &webpage.Facts{}}, nil
}

// Is reports whether the node has the type or a subtype of it.
func (c *Context) Is(n *jsonld.Object, ty string) bool {
	return c.Vocab.Is(jsonld.TypeNames(n), ty)
}

// Single is the node's one known type, or "" when it has none or several.
func (c *Context) Single(n *jsonld.Object) string {
	types := jsonld.TypeNames(n)
	if len(types) != 1 {
		return ""
	}
	if _, ok := c.Vocab.V.Classes[types[0]]; !ok {
		return ""
	}
	return types[0]
}

// Has reports a node of the type, or a subtype, in the draft.
func (c *Context) Has(ty string) bool {
	return slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, ty) })
}

// indexKey is the memo key of the draft's index.
type indexKey struct{}

// Index is the draft's index of values, URLs and node paths, built once.
func (c *Context) Index() *jsonld.Index {
	return Memo(c, indexKey{}, func() *jsonld.Index { return jsonld.NewIndex(c.Nodes) })
}

// Indexed reports a page that search engines index: there is a page, and
// nothing keeps it out of search.
func (c *Context) Indexed() bool { return c.Page != nil && c.Page.Private == "" }

// Memo is the fact stored under key, computed by f on first use. A key is a
// value of a type its owner declares, so owners never collide; every use of a
// key must ask for the same T. f may read other memoized facts, but not the
// one it computes.
func Memo[T any](c *Context, key any, f func() T) T {
	if v, ok := c.memo[key]; ok {
		return v.(T)
	}
	v := f()
	if c.memo == nil {
		c.memo = map[any]any{}
	}
	c.memo[key] = v
	return v
}
