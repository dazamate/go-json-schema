package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(AddToBagForm{}) }

// AddToBagForm warns on an add-to-bag form no Action in the draft targets:
// the Offer needs a BuyAction whose EntryPoint is the form. It needs the
// page's document, on a page search engines index.
type AddToBagForm struct{}

// ID is the check's ID.
func (AddToBagForm) ID() string { return "content/add-to-bag-form" }

// Run reports each such form.
func (AddToBagForm) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range formsOfKind(c, formAddToBag) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's add-to-bag form has no Action whose target points at it: add a BuyAction as the Offer's potentialAction, target an EntryPoint (urlTemplate %s, httpMethod %s, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)", f.action, f.method))
	}
	return out
}
