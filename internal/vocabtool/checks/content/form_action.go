package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(FormAction{}) }

// FormAction warns on a form no Action in the draft targets, other than a
// pre-order, add-to-bag or newsletter form, which have checks of their own.
// The Action says what the form does for a visitor (ask, quote, reserve).
// A search box and a dialog's form are not asked for one. It needs the
// page's document, on a page search engines index.
type FormAction struct{}

// ID is the check's ID.
func (FormAction) ID() string { return "content/form-action" }

// Run reports each such form.
func (FormAction) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range formsOfKind(c, formOther) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's form has no Action whose target points at it: add the Action it performs (AskAction, QuoteAction, ReserveAction, SubscribeAction and the like) %s", f.where))
	}
	return out
}
