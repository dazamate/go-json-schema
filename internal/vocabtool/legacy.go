package vocabtool

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// The legacy adapter runs the checks that have not moved to their own files
// yet, through the old checker, as one audit check. It goes once every check
// has moved.

func init() { audit.Register(legacyAll{}) }

// legacyAll is every check the old checker still makes, in today's order.
type legacyAll struct{}

// ID is the adapter's check ID.
func (legacyAll) ID() string { return "legacy/all" }

// Run runs the old checker's checks over the draft.
func (legacyAll) Run(ctx *audit.Context) []audit.Issue {
	doc, ok := ctx.Root.(*jsonld.Object)
	if !ok {
		return nil
	}
	c := newChecker(ctx.Vocab)
	all := ctx.Nodes
	if ctx.Page != nil {
		c.content(all, ctx.Page)
	}
	c.organizations(all, ctx.In.Keyword)
	c.looseNodes(all, ctx.Known)
	if strings.TrimSpace(ctx.In.Keyword) != "" {
		c.keyword(all, doc, ctx.In.Keyword, ctx.In.KeywordDensity, ctx.Known)
	}
	return c.issues
}
