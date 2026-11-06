package shop

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(PageKindType{}) }

var _ audit.Retyper = PageKindType{}

// PageKindType warns when the page's WebPage is typed more generally than
// the kind of page it is: a contact page's WebPage is a ContactPage. It
// leaves be a node an earlier source in the retype ledger told, or one whose
// name or URL already named the subtype it wants. It needs the page's
// document, on a page search engines index.
type PageKindType struct{}

// ID is the check's ID.
func (PageKindType) ID() string { return "shop/page-kind-type" }

// find is the page's WebPage node and its type, when its kind of page calls
// for a subtype of that type that no source before it named.
func (PageKindType) find(c *audit.Context, before audit.Retyped) (p *pageNode, ty string, ok bool) {
	p = pageNodeOf(c)
	if p == nil || p.node.N == nil {
		return nil, "", false
	}
	n := p.node.N
	ty = c.Single(n)
	if ty == "" || ty == p.want || !slices.Contains(c.Vocab.V.Ancestors(p.want), ty) || before.Told[n] || firstNamed(before.Named[n]) == p.want {
		return nil, "", false
	}
	return p, ty, true
}

// firstNamed is the first subtype the ledger names for a node. The name and
// URL check's message names its subtypes as one list, and the message was the
// only trace of it that the page's kind looked for, so only the first counts:
// "type it AboutPage or ContactPage" leaves the page's kind free to say
// ContactPage.
func firstNamed(named []string) string {
	if len(named) == 0 {
		return ""
	}
	return named[0]
}

// Run reports the WebPage with the type its kind of page takes.
func (k PageKindType) Run(c *audit.Context) []audit.Issue {
	p, ty, ok := k.find(c, c.RetypedBefore(k.ID()))
	if !ok {
		return nil
	}
	return []audit.Issue{audit.Issuef(p.node.Path, audit.Warning, "this %s is a %s page: type it %s", ty, strings.TrimPrefix(string(c.Page.Kind), "kind"), p.want)}
}

// Retypes tells the WebPage and names the type its kind of page takes.
func (k PageKindType) Retypes(c *audit.Context, before audit.Retyped) []audit.Retype {
	p, _, ok := k.find(c, before)
	if !ok {
		return nil
	}
	return []audit.Retype{{Node: p.node.N, Named: []string{p.want}}}
}
