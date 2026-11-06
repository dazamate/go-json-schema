package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(TelLink{}) }

// TelLink warns on a tel: link the page shows when no node in the draft
// has that telephone, on the business or a ContactPoint. It reads the
// page's links, so it needs the page's document.
type TelLink struct{}

// ID is the check's ID.
func (TelLink) ID() string { return "content/tel-link" }

// Run reports each find. It runs on a page search engines index.
func (TelLink) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	return contactIssues(contactsOf(c).telLinks, "the page links %s but no node has that telephone: set telephone on the business or a ContactPoint")
}
