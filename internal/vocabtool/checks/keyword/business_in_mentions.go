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
	var out []audit.Issue
	for _, f := range organizationsOf(c).businessInMentions {
		out = append(out, audit.Issuef(f.path, audit.Warning, "%s is another business in mentions: its node ties the page to that business. Keep it only if it is a partner or the page's subject, never a competitor", f.name))
	}
	return out
}
