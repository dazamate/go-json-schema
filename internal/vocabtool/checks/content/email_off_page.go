package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(EmailOffPage{}) }

// EmailOffPage warns on an email in the markup that the page neither
// writes nor links: only an address the page shows belongs in the markup.
type EmailOffPage struct{}

// ID is the check's ID.
func (EmailOffPage) ID() string { return "content/email-off-page" }

// Run reports each find. It runs on a page search engines index.
func (EmailOffPage) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	return contactIssues(contactsOf(c).emails, "email %s is not on the page: only an address the page shows goes in the markup")
}
