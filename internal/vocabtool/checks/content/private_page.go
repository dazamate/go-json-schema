package content

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(PrivatePage{}) }

// privateNotes say what a page search engines skip would be typed as.
var privateNotes = map[webpage.Kind]string{
	webpage.KindSearch:   "its WebPage is a SearchResultsPage",
	webpage.KindCheckout: "its WebPage is a CheckoutPage",
}

// PrivatePage notes, as information, that search engines leave the page out
// of their index (it is noindex, a cart, a checkout or an account page) and
// so ignore its structured data: the checks of what a page shows ask nothing
// of such a page. It runs on any page that is read.
type PrivatePage struct{}

// ID is the check's ID.
func (PrivatePage) ID() string { return "content/private-page" }

// Run notes the page when search engines skip it.
func (PrivatePage) Run(c *audit.Context) []audit.Issue {
	p := c.Page
	if p == nil || p.Private == "" {
		return nil
	}
	what := "the page is " + p.Private
	if p.Private == "noindex" {
		what = "the page is noindex"
	}
	note := ""
	if n := privateNotes[p.Kind]; n != "" {
		note = "; if you mark it up at all, " + n
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Info, "%s: search engines leave its structured data out, so the page checks skip it%s", what, note)}
}
