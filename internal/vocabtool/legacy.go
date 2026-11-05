package vocabtool

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// The legacy adapters run the checks that have not moved to their own files
// yet, through the old checker, as two audit checks. They go once every
// check has moved.

func init() {
	audit.Register(legacyTerms{})
	audit.Register(legacyAll{})
}

// PageElementsKey is the memo key of the page elements report, the sections
// of the page and the draft nodes that mark them up. Check copies it into
// Report.PageElements.
type PageElementsKey struct{}

// legacyTerms is the vocabulary walk: unknown and superseded terms, enum
// members, and properties used outside their types.
type legacyTerms struct{}

// ID is the walk's check ID.
func (legacyTerms) ID() string { return "terms/legacy" }

// Run walks the draft.
func (legacyTerms) Run(ctx *audit.Context) []audit.Issue {
	c := newChecker(ctx.Vocab)
	c.walk("$", ctx.Root)
	return c.issues
}

// legacyAll is every other check the old checker makes, in today's order.
type legacyAll struct{}

// ID is the adapter's check ID.
func (legacyAll) ID() string { return "legacy/all" }

// Run walks the draft first, since the checks after it read the nodes the
// walk told to take a subtype, then runs the rest and returns only what they
// found: legacyTerms reports the walk's issues.
func (legacyAll) Run(ctx *audit.Context) []audit.Issue {
	doc, ok := ctx.Root.(*jsonld.Object)
	if !ok {
		return nil
	}
	c := newChecker(ctx.Vocab)
	c.walk("$", doc)
	n := len(c.issues)
	all := ctx.Nodes
	c.repeats(doc)
	c.classTypes(all)
	c.namedTypes(all)
	c.webPageElements(all)
	if ctx.HTML != nil {
		// a page search engines skip still lists its sections, unasked
		elements := c.pageElements(all, ctx.HTML, ctx.Page == nil || ctx.Page.Private == "")
		audit.Memo(ctx, PageElementsKey{}, func() []PageElement { return elements })
	}
	text := ""
	if ctx.Page != nil {
		text = ctx.Page.Text
	}
	c.pageEntities(all, text)
	c.bare(all)
	c.ownReviews(all)
	c.reviews(all)
	c.productSnippets(all)
	if ctx.Page != nil {
		c.content(all, ctx.Page)
	}
	c.organizations(all, ctx.In.Keyword)
	c.looseNodes(all, ctx.Known)
	if strings.TrimSpace(ctx.In.Keyword) != "" {
		c.keyword(all, doc, ctx.In.Keyword, ctx.In.KeywordDensity, ctx.Known)
	}
	return c.issues[n:]
}
