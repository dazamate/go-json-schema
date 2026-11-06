package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(NewsletterForm{}) }

// NewsletterForm warns on a newsletter sign-up form no Action in the draft
// targets: the business needs a SubscribeAction whose EntryPoint is the form.
// It needs the page's document, on a page search engines index.
type NewsletterForm struct{}

// ID is the check's ID.
func (NewsletterForm) ID() string { return "content/newsletter-form" }

// Run reports each such form.
func (NewsletterForm) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range formsOfKind(c, formNewsletter) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the page's newsletter form has no Action whose target points at it: add a SubscribeAction (to the business) %s", f.where))
	}
	return out
}
