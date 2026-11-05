package keyword

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(BusinessInMentions{}) }

// BusinessInMentions warns about another business in the page's mentions: its
// node ties the page to that business, so keep it only for a partner or the
// page's subject, never a competitor. The site's own business is exempt, and
// so is a business the keyword itself names. It reads no page.
type BusinessInMentions struct{}

// ID is the check's ID.
func (BusinessInMentions) ID() string { return "keyword/business-in-mentions" }

// Run reports each other business in mentions.
func (BusinessInMentions) Run(c *audit.Context) []audit.Issue {
	return organizationsOf(c).businessInMentions
}
