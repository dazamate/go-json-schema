package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(TelephoneOffPage{}) }

// TelephoneOffPage warns on a telephone in the markup that the page
// neither writes nor links: only a number the page shows belongs in the
// markup.
type TelephoneOffPage struct{}

// ID is the check's ID.
func (TelephoneOffPage) ID() string { return "content/telephone-off-page" }

// Run reports each find. It runs on a page search engines index.
func (TelephoneOffPage) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	return contactIssues(contactsOf(c).telephones, "telephone %s is not on the page: only a number the page shows goes in the markup")
}
