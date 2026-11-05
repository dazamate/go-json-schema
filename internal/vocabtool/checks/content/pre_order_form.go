package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(PreOrderForm{}) }

// PreOrderForm warns on a pre-order form no Action in the draft targets: the
// Offer needs a PreOrderAction whose EntryPoint is the form. It needs the
// page's document, on a page search engines index.
type PreOrderForm struct{}

// ID is the check's ID.
func (PreOrderForm) ID() string { return "content/pre-order-form" }

// Run reports each such form.
func (PreOrderForm) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range formsOfKind(c, formPreOrder) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's pre-order form has no Action whose target points at it: add a PreOrderAction as the Offer's potentialAction, target an EntryPoint (urlTemplate %s, httpMethod %s)", f.action, f.method))
	}
	return out
}
