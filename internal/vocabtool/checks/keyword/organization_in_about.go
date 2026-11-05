package keyword

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(OrganizationInAbout{}) }

// OrganizationInAbout warns about an organization that is no business in the
// page's about: a page is about its topic, not an agency or source it names,
// so the organization belongs in mentions. The site's own business is exempt,
// and so is an organization the keyword itself names. It reads no page.
type OrganizationInAbout struct{}

// ID is the check's ID.
func (OrganizationInAbout) ID() string { return "keyword/organization-in-about" }

// Run reports each organization in about that is no business.
func (OrganizationInAbout) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range organizationsOf(c).organizationAbout {
		out = append(out, audit.Issuef(f.path, audit.Warning, "%s is an organization in about: a page is about its topic, not an agency or source it names; move it to mentions", f.name))
	}
	return out
}
