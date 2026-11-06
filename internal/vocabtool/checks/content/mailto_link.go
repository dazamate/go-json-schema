package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(MailtoLink{}) }

// MailtoLink warns on a mailto: link the page shows when no node in the
// draft has that email, on the business or a ContactPoint. It reads the
// page's links, so it needs the page's document.
type MailtoLink struct{}

// ID is the check's ID.
func (MailtoLink) ID() string { return "content/mailto-link" }

// Run reports each find. It runs on a page search engines index.
func (MailtoLink) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	return contactIssues(contactsOf(c).mailLinks, "the page links %s but no node has that email: set email on the business or a ContactPoint")
}
